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

// ConvertPointHandler 处理点数兑换
type ConvertPointHandler struct {
	Db            *sql.DB
	Logger        *zap.Logger
	ConvertNumber int
}

// GetType 可以处理的消息类型
func (*ConvertPointHandler) GetType() byte {
	return 0xE1
}

// GetResponse 根据请求获得响应
func (h *ConvertPointHandler) GetResponse(request *common.BillingPacket) *common.BillingPacket {
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
	//orderId 21u
	orderIDBytes := packetReader.ReadBytes(21)
	mGoodsTypeNum := packetReader.ReadUint16() //始终为1
	//物品类型
	mGoodsType := packetReader.ReadInt()
	//物品数量
	mGoodsNumber := packetReader.ReadUint16()
	//获取需要兑换的点数:4u
	needPoint := packetReader.ReadInt()
	needPoint /= h.ConvertNumber
	if needPoint < 0 {
		needPoint = 0
	}
	//每次兑换点数上限 u2
	var maxPoint = 0xffff //65535
	if needPoint > maxPoint {
		needPoint = maxPoint
	}
	userPoint := 0
	//获取用户当前点数总额
	account, err := models.GetAccountByUsername(h.Db, string(username))
	if err != nil {
		h.Logger.Error("get account:" + string(username) + " info failed: " + err.Error())
	}
	if account != nil {
		userPoint = account.Point
		if userPoint < 0 {
			userPoint = 0
		}
	}
	//最终可兑换的点数
	var realPoint int
	if needPoint > userPoint {
		realPoint = userPoint
	} else {
		realPoint = needPoint
	}
	// 执行兑换
	err = models.ConvertUserPoint(h.Db, string(username), realPoint)
	if err != nil {
		h.Logger.Error("convert point failed: " + err.Error())
		realPoint = 0
	} else {
		h.Logger.Info(fmt.Sprintf("user [%s] %s(ip: %s) point total [%d], need point [%d]: %d-%d=%d",
			username, charName, loginIP, userPoint, needPoint,
			userPoint, realPoint, userPoint-realPoint))
	}
	// 数据包组合
	var opData []byte
	opData = append(opData, usernameLength)
	opData = append(opData, username...)
	opData = append(opData, orderIDBytes...)
	opData = append(opData, 0x00)
	//写入剩余点数
	leftPoint := (userPoint - realPoint) * h.ConvertNumber
	for i := 0; i < 4; i++ {
		tmpValue := leftPoint
		movePos := (3 - i) * 8
		if movePos > 0 {
			tmpValue >>= movePos
		}
		opData = append(opData, byte(tmpValue&0xff))
	}
	//mGoodsTypeNum
	opData = append(opData, byte((mGoodsTypeNum&0xff00)>>8), byte(mGoodsTypeNum&0xff))
	// 写入mGoodsType
	for i := 0; i < 4; i++ {
		tmpValue := mGoodsType
		movePos := (3 - i) * 8
		if movePos > 0 {
			tmpValue >>= movePos
		}
		opData = append(opData, byte(tmpValue&0xff))
	}
	//写入mGoodsNumber(购买的数量)
	opData = append(opData, byte((mGoodsNumber&0xff00)>>8), byte(mGoodsNumber&0xff))
	response.OpData = opData
	return response
}
