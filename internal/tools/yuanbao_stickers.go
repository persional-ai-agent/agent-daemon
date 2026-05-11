package tools

import (
	"sort"
	"strings"
)

type yuanbaoSticker struct {
	StickerID   string `json:"sticker_id"`
	PackageID   string `json:"package_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Minimal built-in Yuanbao sticker catalogue (subset).
// Hermes has a much larger built-in list; expand as needed.
var yuanbaoStickers = []yuanbaoSticker{
	{StickerID: "278", PackageID: "1003", Name: "六六六", Description: "666 厉害 牛 棒 绝了 好强 awesome"},
	{StickerID: "252", PackageID: "1003", Name: "比心", Description: "笔芯 爱你 爱心手势 love heart 喜欢你"},
	{StickerID: "222", PackageID: "1003", Name: "吃瓜", Description: "围观 看戏 八卦 路人 看热闹 板凳"},
	{StickerID: "225", PackageID: "1003", Name: "狗头", Description: "doge 保命 开玩笑 滑稽 反讽 懂的都懂"},
	{StickerID: "131", PackageID: "1003", Name: "酷", Description: "帅 墨镜 cool 高冷 有型 swagger"},
	{StickerID: "151", PackageID: "1003", Name: "奋斗", Description: "努力 加油 拼搏 冲 干劲 卷起来"},
	{StickerID: "199", PackageID: "1003", Name: "泪奔", Description: "大哭 伤心 破防 感动哭 泪流满面 呜呜"},
	{StickerID: "246", PackageID: "1003", Name: "打call", Description: "应援 加油 支持 喝彩 助威 call"},
	{StickerID: "248", PackageID: "1003", Name: "仔细分析", Description: "思考 推敲 认真 研究 琢磨 让我想想"},
	{StickerID: "130", PackageID: "1003", Name: "害羞", Description: "腼腆 不好意思 脸红 娇羞 羞涩 捂脸"},
}

func searchYuanbaoStickers(query string, limit int) []yuanbaoSticker {
	q := strings.ToLower(strings.TrimSpace(query))
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	type scored struct {
		s     yuanbaoSticker
		score int
	}
	scoredList := make([]scored, 0, len(yuanbaoStickers))
	for _, s := range yuanbaoStickers {
		name := strings.ToLower(s.Name)
		desc := strings.ToLower(s.Description)
		score := 0
		if q == "" {
			score = 1
		} else {
			if strings.Contains(name, q) {
				score += 5
			}
			if strings.Contains(desc, q) {
				score += 2
			}
		}
		if score > 0 {
			scoredList = append(scoredList, scored{s: s, score: score})
		}
	}
	sort.SliceStable(scoredList, func(i, j int) bool {
		if scoredList[i].score == scoredList[j].score {
			return scoredList[i].s.Name < scoredList[j].s.Name
		}
		return scoredList[i].score > scoredList[j].score
	})
	out := make([]yuanbaoSticker, 0, limit)
	for _, it := range scoredList {
		out = append(out, it.s)
		if len(out) >= limit {
			break
		}
	}
	return out
}

