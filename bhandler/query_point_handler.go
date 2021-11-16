package bhandler

import (
	"database/sql"
	"fmt"
	"github.com/liuguangw/billing_go/common"
	"github.com/liuguangw/billing_go/models"
	"github.com/liuguangw/billing_go/services"
	"go.uber.org/zap"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// QueryPointHandler 查询点数
type QueryPointHandler struct {
	Db          *sql.DB
	Logger      *zap.Logger
	LoginUsers  map[string]*common.ClientInfo //已登录,还未进入游戏的用户
	OnlineUsers map[string]*common.ClientInfo //已进入游戏的用户
	MacCounters map[string]int                //已进入游戏的用户的mac地址计数器
}

// GetType 可以处理的消息类型
func (*QueryPointHandler) GetType() byte {
	return packetTypeQueryPoint
}

// GetResponse 根据请求获得响应
func (h *QueryPointHandler) GetResponse(request *common.BillingPacket) *common.BillingPacket {
	response := request.PrepareResponse()
	packetReader := services.NewPacketDataReader(request.OpData)
	//用户名
	usernameLength := packetReader.ReadByteValue()
	tmpLength := int(usernameLength)
	username := packetReader.ReadBytes(tmpLength)
	//登录IP
	tmpLength = int(packetReader.ReadByteValue())
	loginIP := string(packetReader.ReadBytes(tmpLength))
	//角色名
	tmpLength = int(packetReader.ReadByteValue())
	charNameGbkData := packetReader.ReadBytes(tmpLength)
	gbkDecoder := simplifiedchinese.GBK.NewDecoder()
	charName, err := gbkDecoder.Bytes(charNameGbkData)
	if err != nil {
		h.Logger.Error("decode char name failed: " + err.Error())
		charName = []byte("?")
	}
	account, err := models.GetAccountByUsername(h.Db, string(username))
	if err != nil {
		h.Logger.Error("get account:" + string(username) + " info failed: " + err.Error())
	}
	//标记在线
	clientInfo := &common.ClientInfo{
		IP:       loginIP,
		CharName: string(charName),
	}
	markOnline(h.LoginUsers, h.OnlineUsers, h.MacCounters, string(username), clientInfo)
	//
	var accountPoint = 0
	if account != nil {
		accountPoint = (account.Point + 1) * 1000
	}
	h.Logger.Info(fmt.Sprintf("user [%s] %s query point (%d) at %s", username, charName, account.Point, loginIP))
	var opData []byte
	opData = append(opData, usernameLength)
	opData = append(opData, username...)
	for i := 0; i < 4; i++ {
		tmpValue := accountPoint
		movePos := (3 - i) * 8
		if movePos > 0 {
			tmpValue >>= movePos
		}
		opData = append(opData, byte(tmpValue&0xff))
	}
	response.OpData = opData
	return response
}
