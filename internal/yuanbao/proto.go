package yuanbao

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"
)

// Minimal Yuanbao WS protobuf codec ported from Hermes:
// /data/source/hermes-agent/gateway/platforms/yuanbao_proto.py

const (
	wireVarint = 0
	wireLen    = 2
)

const (
	CmdTypeRequest  = 0
	CmdTypeResponse = 1
	CmdTypePush     = 2
	CmdTypePushAck  = 3
)

const (
	connModule = "conn_access"
	cmdAuthBind = "auth-bind"
	cmdPing     = "ping"
)

const (
	bizPkg = "yuanbao_openclaw_proxy"
)

const HermesInstanceID = 17

type Head struct {
	CmdType  uint32
	Cmd      string
	SeqNo    uint32
	MsgID    string
	Module   string
	NeedAck  bool
	Status   uint64
}

type ConnMsg struct {
	Head Head
	Data []byte
}

var seqCounter uint32

func NextSeqNo() uint32 {
	return atomic.AddUint32(&seqCounter, 1) - 1
}

func encodeVarint(v uint64) []byte {
	var buf [10]byte
	n := binary.PutUvarint(buf[:], v)
	return buf[:n]
}

func decodeVarint(b []byte, pos int) (uint64, int, error) {
	var x uint64
	var s uint
	for i := 0; i < 10 && pos < len(b); i++ {
		c := b[pos]
		pos++
		if c < 0x80 {
			if i == 9 && c > 1 {
				return 0, 0, errors.New("varint overflow")
			}
			return x | uint64(c)<<s, pos, nil
		}
		x |= uint64(c&0x7f) << s
		s += 7
	}
	return 0, 0, errors.New("truncated varint")
}

func encodeField(fieldNumber int, wireType int, value []byte) []byte {
	tag := uint64((fieldNumber << 3) | wireType)
	out := make([]byte, 0, 10+len(value))
	out = append(out, encodeVarint(tag)...)
	out = append(out, value...)
	return out
}

func encodeString(s string) []byte {
	b := []byte(s)
	out := make([]byte, 0, 10+len(b))
	out = append(out, encodeVarint(uint64(len(b)))...)
	out = append(out, b...)
	return out
}

func encodeBytes(b []byte) []byte {
	out := make([]byte, 0, 10+len(b))
	out = append(out, encodeVarint(uint64(len(b)))...)
	out = append(out, b...)
	return out
}

func encodeMessage(b []byte) []byte { return encodeBytes(b) }

type fieldEntry struct {
	wire int
	v    any
}

func parseFields(b []byte) (map[int][]fieldEntry, error) {
	pos := 0
	out := map[int][]fieldEntry{}
	for pos < len(b) {
		tag, p2, err := decodeVarint(b, pos)
		if err != nil {
			return nil, err
		}
		pos = p2
		fn := int(tag >> 3)
		wt := int(tag & 0x7)
		switch wt {
		case wireVarint:
			val, p3, err := decodeVarint(b, pos)
			if err != nil {
				return nil, err
			}
			pos = p3
			out[fn] = append(out[fn], fieldEntry{wire: wt, v: val})
		case wireLen:
			l, p3, err := decodeVarint(b, pos)
			if err != nil {
				return nil, err
			}
			pos = p3
			if pos+int(l) > len(b) {
				return nil, errors.New("length-delimited out of range")
			}
			val := b[pos : pos+int(l)]
			pos += int(l)
			cp := make([]byte, len(val))
			copy(cp, val)
			out[fn] = append(out[fn], fieldEntry{wire: wt, v: cp})
		default:
			return nil, fmt.Errorf("unsupported wire type %d", wt)
		}
	}
	return out, nil
}

func getFirstBytes(fields map[int][]fieldEntry, fn int) []byte {
	entries := fields[fn]
	if len(entries) == 0 {
		return nil
	}
	if entries[0].wire != wireLen {
		return nil
	}
	b, _ := entries[0].v.([]byte)
	return b
}

func getFirstString(fields map[int][]fieldEntry, fn int) string {
	b := getFirstBytes(fields, fn)
	if b == nil {
		return ""
	}
	return string(b)
}

func getFirstVarint(fields map[int][]fieldEntry, fn int) uint64 {
	entries := fields[fn]
	if len(entries) == 0 {
		return 0
	}
	if entries[0].wire != wireVarint {
		return 0
	}
	v, _ := entries[0].v.(uint64)
	return v
}

func encodeHead(h Head) []byte {
	buf := make([]byte, 0, 128)
	if h.CmdType != 0 {
		buf = append(buf, encodeField(1, wireVarint, encodeVarint(uint64(h.CmdType)))...)
	}
	if h.Cmd != "" {
		buf = append(buf, encodeField(2, wireLen, encodeString(h.Cmd))...)
	}
	if h.SeqNo != 0 {
		buf = append(buf, encodeField(3, wireVarint, encodeVarint(uint64(h.SeqNo)))...)
	}
	if h.MsgID != "" {
		buf = append(buf, encodeField(4, wireLen, encodeString(h.MsgID))...)
	}
	if h.Module != "" {
		buf = append(buf, encodeField(5, wireLen, encodeString(h.Module))...)
	}
	if h.NeedAck {
		buf = append(buf, encodeField(6, wireVarint, encodeVarint(1))...)
	}
	if h.Status != 0 {
		buf = append(buf, encodeField(10, wireVarint, encodeVarint(h.Status))...)
	}
	return buf
}

func decodeHead(b []byte) (Head, error) {
	f, err := parseFields(b)
	if err != nil {
		return Head{}, err
	}
	return Head{
		CmdType: uint32(getFirstVarint(f, 1)),
		Cmd:     getFirstString(f, 2),
		SeqNo:   uint32(getFirstVarint(f, 3)),
		MsgID:   getFirstString(f, 4),
		Module:  getFirstString(f, 5),
		NeedAck: getFirstVarint(f, 6) != 0,
		Status:  getFirstVarint(f, 10),
	}, nil
}

func EncodeConnMsgFull(h Head, data []byte) []byte {
	headBytes := encodeHead(h)
	buf := make([]byte, 0, 16+len(headBytes)+len(data))
	buf = append(buf, encodeField(1, wireLen, encodeMessage(headBytes))...)
	if len(data) > 0 {
		buf = append(buf, encodeField(2, wireLen, encodeBytes(data))...)
	}
	return buf
}

func DecodeConnMsg(b []byte) (ConnMsg, error) {
	f, err := parseFields(b)
	if err != nil {
		return ConnMsg{}, err
	}
	headBytes := getFirstBytes(f, 1)
	dataBytes := getFirstBytes(f, 2)
	h, err := decodeHead(headBytes)
	if err != nil {
		return ConnMsg{}, err
	}
	return ConnMsg{Head: h, Data: dataBytes}, nil
}

func EncodeBizMsg(method, reqID string, body []byte) []byte {
	return EncodeConnMsgFull(Head{
		CmdType: CmdTypeRequest,
		Cmd:     method,
		SeqNo:   NextSeqNo(),
		MsgID:   reqID,
		Module:  bizPkg,
	}, body)
}

func EncodeAuthBind(uid, source, token, msgID, appVersion, operationSystem, botVersion, routeEnv string) []byte {
	// AuthInfo: uid=1, source=2, token=3
	authBuf := make([]byte, 0, 128)
	authBuf = append(authBuf, encodeField(1, wireLen, encodeString(uid))...)
	authBuf = append(authBuf, encodeField(2, wireLen, encodeString(source))...)
	authBuf = append(authBuf, encodeField(3, wireLen, encodeString(token))...)

	// DeviceInfo: app_version=1, app_operation_system=2, instance_id=10, bot_version=24
	devBuf := make([]byte, 0, 128)
	if appVersion != "" {
		devBuf = append(devBuf, encodeField(1, wireLen, encodeString(appVersion))...)
	}
	if operationSystem != "" {
		devBuf = append(devBuf, encodeField(2, wireLen, encodeString(operationSystem))...)
	}
	devBuf = append(devBuf, encodeField(10, wireLen, encodeString(fmt.Sprintf("%d", HermesInstanceID)))...)
	if botVersion != "" {
		devBuf = append(devBuf, encodeField(24, wireLen, encodeString(botVersion))...)
	}

	// AuthBindReq: biz_id=1, auth_info=2, device_info=3, env_name=5
	reqBuf := make([]byte, 0, 256)
	reqBuf = append(reqBuf, encodeField(1, wireLen, encodeString("ybBot"))...)
	reqBuf = append(reqBuf, encodeField(2, wireLen, encodeMessage(authBuf))...)
	reqBuf = append(reqBuf, encodeField(3, wireLen, encodeMessage(devBuf))...)
	if routeEnv != "" {
		reqBuf = append(reqBuf, encodeField(5, wireLen, encodeString(routeEnv))...)
	}

	return EncodeConnMsgFull(Head{
		CmdType: CmdTypeRequest,
		Cmd:     cmdAuthBind,
		SeqNo:   NextSeqNo(),
		MsgID:   msgID,
		Module:  connModule,
	}, reqBuf)
}

func EncodePing(msgID string) []byte {
	return EncodeConnMsgFull(Head{
		CmdType: CmdTypeRequest,
		Cmd:     cmdPing,
		SeqNo:   NextSeqNo(),
		MsgID:   msgID,
		Module:  connModule,
	}, nil)
}

func EncodePushAck(original Head) []byte {
	return EncodeConnMsgFull(Head{
		CmdType: CmdTypePushAck,
		Cmd:     original.Cmd,
		SeqNo:   NextSeqNo(),
		MsgID:   original.MsgID,
		Module:  original.Module,
	}, nil)
}

// ---- Biz payload encoders (minimal) ----

func EncodeMsgContentText(text string) []byte {
	buf := make([]byte, 0, len(text)+16)
	if text != "" {
		buf = append(buf, encodeField(1, wireLen, encodeString(text))...)
	}
	return buf
}

func EncodeMsgContentFace(index uint32, data string) []byte {
	buf := make([]byte, 0, len(data)+32)
	if data != "" {
		buf = append(buf, encodeField(4, wireLen, encodeString(data))...)
	}
	if index != 0 {
		buf = append(buf, encodeField(9, wireVarint, encodeVarint(uint64(index)))...)
	} else {
		// Yuanbao convention: still include index=0 for TIMFaceElem.
		buf = append(buf, encodeField(9, wireVarint, encodeVarint(0))...)
	}
	return buf
}

func EncodeMsgBodyElement(msgType string, msgContent []byte) []byte {
	buf := make([]byte, 0, 64+len(msgType)+len(msgContent))
	if msgType != "" {
		buf = append(buf, encodeField(1, wireLen, encodeString(msgType))...)
	}
	if len(msgContent) > 0 {
		buf = append(buf, encodeField(2, wireLen, encodeMessage(msgContent))...)
	}
	return buf
}

func EncodeSendC2CMessageReq(toAccount, fromAccount string, msgBody [][]byte) []byte {
	// Fields:
	// 2: to_account (string)
	// 3: from_account (string, optional)
	// 5: msg_body (repeated MsgBodyElement)
	buf := make([]byte, 0, 256)
	buf = append(buf, encodeField(2, wireLen, encodeString(toAccount))...)
	if fromAccount != "" {
		buf = append(buf, encodeField(3, wireLen, encodeString(fromAccount))...)
	}
	for _, el := range msgBody {
		buf = append(buf, encodeField(5, wireLen, encodeMessage(el))...)
	}
	return buf
}

func EncodeSendGroupMessageReq(groupCode, fromAccount, refMsgID string, msgBody [][]byte) []byte {
	// Fields:
	// 2: group_code (string)
	// 3: from_account (string, optional)
	// 6: msg_body (repeated MsgBodyElement)
	// 7: ref_msg_id (string, optional)
	buf := make([]byte, 0, 256)
	buf = append(buf, encodeField(2, wireLen, encodeString(groupCode))...)
	if fromAccount != "" {
		buf = append(buf, encodeField(3, wireLen, encodeString(fromAccount))...)
	}
	for _, el := range msgBody {
		buf = append(buf, encodeField(6, wireLen, encodeMessage(el))...)
	}
	if refMsgID != "" {
		buf = append(buf, encodeField(7, wireLen, encodeString(refMsgID))...)
	}
	return buf
}

func EncodeQueryGroupInfoReq(groupCode string) []byte {
	return encodeField(1, wireLen, encodeString(groupCode))
}

func EncodeGetGroupMemberListReq(groupCode string, offset, limit uint32) []byte {
	buf := make([]byte, 0, 64)
	buf = append(buf, encodeField(1, wireLen, encodeString(groupCode))...)
	if offset != 0 {
		buf = append(buf, encodeField(2, wireVarint, encodeVarint(uint64(offset)))...)
	}
	buf = append(buf, encodeField(3, wireVarint, encodeVarint(uint64(limit)))...)
	return buf
}

func DecodeAuthBindRsp(data []byte) (code int, message, connectID string, err error) {
	f, err := parseFields(data)
	if err != nil {
		return 0, "", "", err
	}
	code = int(getFirstVarint(f, 1))
	message = getFirstString(f, 2)
	connectID = getFirstString(f, 3)
	return code, message, connectID, nil
}

func DecodeQueryGroupInfoRsp(data []byte) (map[string]any, error) {
	f, err := parseFields(data)
	if err != nil {
		return nil, err
	}
	code := int(getFirstVarint(f, 1))
	msg := getFirstString(f, 2)
	res := map[string]any{"code": code}
	if msg != "" {
		res["message"] = msg
	}
	giBytes := getFirstBytes(f, 3)
	if len(giBytes) > 0 {
		gi, err := parseFields(giBytes)
		if err == nil {
			res["group_name"] = getFirstString(gi, 1)
			res["owner_id"] = getFirstString(gi, 2)
			res["owner_nickname"] = getFirstString(gi, 3)
			res["member_count"] = int(getFirstVarint(gi, 4))
		}
	}
	return res, nil
}

func DecodeGetGroupMemberListRsp(data []byte) (map[string]any, error) {
	f, err := parseFields(data)
	if err != nil {
		return nil, err
	}
	code := int(getFirstVarint(f, 1))
	msg := getFirstString(f, 2)
	members := make([]map[string]any, 0)
	for _, entry := range f[3] {
		if entry.wire != wireLen {
			continue
		}
		mb, _ := entry.v.([]byte)
		md, err := parseFields(mb)
		if err != nil {
			continue
		}
		m := map[string]any{
			"user_id":   getFirstString(md, 1),
			"nickname":  getFirstString(md, 2),
			"role":      int(getFirstVarint(md, 3)),
			"join_time": int(getFirstVarint(md, 4)),
			"name_card": getFirstString(md, 5),
		}
		members = append(members, m)
	}
	return map[string]any{
		"code":        code,
		"message":     msg,
		"members":     members,
		"next_offset": int(getFirstVarint(f, 4)),
		"is_complete": getFirstVarint(f, 5) != 0,
	}, nil
}

