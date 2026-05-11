package yuanbao

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

type Client struct {
	wsURL     string
	token     string
	botID     string
	routeEnv  string

	appVersion      string
	operationSystem string
	botVersion      string

	mu       sync.Mutex
	conn     *websocket.Conn
	pending  map[string]chan ConnMsg
	closed   bool
}

type ClientOptions struct {
	WSURL     string
	Token     string
	BotID     string
	RouteEnv  string

	AppVersion      string
	OperationSystem string
	BotVersion      string
}

func NewClient(opts ClientOptions) (*Client, error) {
	if strings.TrimSpace(opts.WSURL) == "" {
		return nil, errors.New("ws_url required")
	}
	if strings.TrimSpace(opts.Token) == "" {
		return nil, errors.New("token required")
	}
	if strings.TrimSpace(opts.BotID) == "" {
		return nil, errors.New("bot_id required")
	}
	c := &Client{
		wsURL:            opts.WSURL,
		token:            opts.Token,
		botID:            opts.BotID,
		routeEnv:         opts.RouteEnv,
		appVersion:       opts.AppVersion,
		operationSystem:  opts.OperationSystem,
		botVersion:       opts.BotVersion,
		pending:          make(map[string]chan ConnMsg),
	}
	return c, nil
}

func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("yuanbao client closed")
	}
	if c.conn != nil {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, c.wsURL, nil)
	if err != nil {
		return err
	}
	conn.SetReadLimit(8 << 20)

	// AUTH_BIND
	msgID := uuid.NewString()
	authBytes := EncodeAuthBind(
		c.botID,
		"bot",
		c.token,
		msgID,
		c.appVersion,
		c.operationSystem,
		c.botVersion,
		c.routeEnv,
	)
	if err := conn.WriteMessage(websocket.BinaryMessage, authBytes); err != nil {
		_ = conn.Close()
		return err
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(12 * time.Second)
	}
	_ = conn.SetReadDeadline(deadline)
	for {
		mt, data, err := conn.ReadMessage()
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("auth-bind read: %w", err)
		}
		if mt != websocket.BinaryMessage {
			continue
		}
		msg, err := DecodeConnMsg(data)
		if err != nil {
			continue
		}
		if msg.Head.CmdType == CmdTypeResponse && msg.Head.Cmd == cmdAuthBind {
			code, m, _, err := DecodeAuthBindRsp(msg.Data)
			if err != nil {
				_ = conn.Close()
				return err
			}
			if code != 0 {
				_ = conn.Close()
				return fmt.Errorf("auth-bind failed: code=%d message=%s", code, m)
			}
			break
		}
	}
	_ = conn.SetReadDeadline(time.Time{})

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	go c.recvLoop()
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	c.closed = true
	conn := c.conn
	c.conn = nil
	for k, ch := range c.pending {
		close(ch)
		delete(c.pending, k)
	}
	c.mu.Unlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *Client) recvLoop() {
	for {
		c.mu.Lock()
		conn := c.conn
		closed := c.closed
		c.mu.Unlock()
		if conn == nil || closed {
			return
		}
		mt, data, err := conn.ReadMessage()
		if err != nil {
			_ = c.Close()
			return
		}
		if mt != websocket.BinaryMessage {
			continue
		}
		msg, err := DecodeConnMsg(data)
		if err != nil {
			continue
		}
		// Handle pushes with need_ack.
		if msg.Head.CmdType == CmdTypePush && msg.Head.NeedAck {
			ack := EncodePushAck(msg.Head)
			_ = conn.WriteMessage(websocket.BinaryMessage, ack)
		}

		c.mu.Lock()
		ch := c.pending[msg.Head.MsgID]
		c.mu.Unlock()
		if ch != nil {
			select {
			case ch <- msg:
			default:
			}
		}
	}
}

func (c *Client) call(ctx context.Context, method string, body []byte) (ConnMsg, error) {
	c.mu.Lock()
	conn := c.conn
	if conn == nil {
		c.mu.Unlock()
		return ConnMsg{}, errors.New("yuanbao not connected")
	}
	reqID := fmt.Sprintf("go_%d", NextSeqNo())
	ch := make(chan ConnMsg, 1)
	c.pending[reqID] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, reqID)
		c.mu.Unlock()
	}()

	msgBytes := EncodeBizMsg(method, reqID, body)
	if err := conn.WriteMessage(websocket.BinaryMessage, msgBytes); err != nil {
		return ConnMsg{}, err
	}

	select {
	case <-ctx.Done():
		return ConnMsg{}, ctx.Err()
	case res, ok := <-ch:
		if !ok {
			return ConnMsg{}, errors.New("yuanbao connection closed")
		}
		return res, nil
	case <-time.After(12 * time.Second):
		return ConnMsg{}, errors.New("yuanbao request timeout")
	}
}

func (c *Client) SendC2C(ctx context.Context, toAccount, text string) (map[string]any, error) {
	el := EncodeMsgBodyElement("TIMTextElem", EncodeMsgContentText(text))
	body := EncodeSendC2CMessageReq(toAccount, c.botID, [][]byte{el})
	res, err := c.call(ctx, "send_c2c_message", body)
	if err != nil {
		return nil, err
	}
	// Best-effort decode: most impls return {code,message}; keep raw bytes too.
	out := map[string]any{"success": res.Head.CmdType == CmdTypeResponse, "req_id": res.Head.MsgID}
	if len(res.Data) > 0 {
		out["raw"] = fmt.Sprintf("%x", res.Data)
	}
	return out, nil
}

func (c *Client) SendGroupText(ctx context.Context, groupCode, text, refMsgID string) (map[string]any, error) {
	el := EncodeMsgBodyElement("TIMTextElem", EncodeMsgContentText(text))
	body := EncodeSendGroupMessageReq(groupCode, c.botID, refMsgID, [][]byte{el})
	res, err := c.call(ctx, "send_group_message", body)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"success": res.Head.CmdType == CmdTypeResponse, "req_id": res.Head.MsgID}
	if len(res.Data) > 0 {
		out["raw"] = fmt.Sprintf("%x", res.Data)
	}
	return out, nil
}

func (c *Client) SendGroupSticker(ctx context.Context, groupCode, stickerJSON, refMsgID string) (map[string]any, error) {
	el := EncodeMsgBodyElement("TIMFaceElem", EncodeMsgContentFace(0, stickerJSON))
	body := EncodeSendGroupMessageReq(groupCode, c.botID, refMsgID, [][]byte{el})
	res, err := c.call(ctx, "send_group_message", body)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"success": res.Head.CmdType == CmdTypeResponse, "req_id": res.Head.MsgID}
	if len(res.Data) > 0 {
		out["raw"] = fmt.Sprintf("%x", res.Data)
	}
	return out, nil
}

func (c *Client) SendC2CSticker(ctx context.Context, toAccount, stickerJSON string) (map[string]any, error) {
	el := EncodeMsgBodyElement("TIMFaceElem", EncodeMsgContentFace(0, stickerJSON))
	body := EncodeSendC2CMessageReq(toAccount, c.botID, [][]byte{el})
	res, err := c.call(ctx, "send_c2c_message", body)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"success": res.Head.CmdType == CmdTypeResponse, "req_id": res.Head.MsgID}
	if len(res.Data) > 0 {
		out["raw"] = fmt.Sprintf("%x", res.Data)
	}
	return out, nil
}

func (c *Client) QueryGroupInfo(ctx context.Context, groupCode string) (map[string]any, error) {
	body := EncodeQueryGroupInfoReq(groupCode)
	res, err := c.call(ctx, "query_group_info", body)
	if err != nil {
		return nil, err
	}
	decoded, derr := DecodeQueryGroupInfoRsp(res.Data)
	if derr != nil {
		return map[string]any{"success": false, "error": derr.Error()}, nil
	}
	decoded["success"] = true
	return decoded, nil
}

func (c *Client) GetGroupMemberList(ctx context.Context, groupCode string, offset, limit uint32) (map[string]any, error) {
	if limit == 0 {
		limit = 200
	}
	body := EncodeGetGroupMemberListReq(groupCode, offset, limit)
	res, err := c.call(ctx, "get_group_member_list", body)
	if err != nil {
		return nil, err
	}
	decoded, derr := DecodeGetGroupMemberListRsp(res.Data)
	if derr != nil {
		return map[string]any{"success": false, "error": derr.Error()}, nil
	}
	decoded["success"] = true
	return decoded, nil
}

