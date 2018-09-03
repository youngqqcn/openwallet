/*
 * Copyright 2018 The OpenWallet Authors
 * This file is part of the OpenWallet library.
 *
 * The OpenWallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The OpenWallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */

package merchant

import (
	"fmt"
	"github.com/blocktree/OpenWallet/assets"
	"github.com/blocktree/OpenWallet/common"
	"github.com/blocktree/OpenWallet/log"
	"github.com/blocktree/OpenWallet/openwallet"
	"github.com/blocktree/OpenWallet/owtp"
	"github.com/pkg/errors"
)

const (

	//订阅类型，1：钱包余额，2：充值记录，3：汇总日志
	SubscribeTypeBalance    = 1
	SubscribeTypeCharge     = 2
	SubscribeTypeSummaryLog = 3
)

var (
	//商户节点
	merchantNode *MerchantNode

	/* 异常错误 */

	//节点断开
	ErrMerchantNodeDisconnected = errors.New("Merchant node is not connected!")
)

/********** 钱包管理相关方法【被动】 **********/

//setupRouter 配置路由
func (m *MerchantNode) setupRouter() {

	m.Node.HandleFunc("subscribe", m.subscribe)
	m.Node.HandleFunc("createWallet", m.createWallet)
	m.Node.HandleFunc("createAddress", m.createAddress)
	m.Node.HandleFunc("getAddressList", m.getAddressList)
	m.Node.HandleFunc("configWallet", m.configWallet)
	m.Node.HandleFunc("getWalletList", m.getWalletList)
	m.Node.HandleFunc("submitTransaction", m.submitTransaction)
	m.Node.HandleFunc("getNewHeight", m.getNewHeight)
	m.Node.HandleFunc("getBalanceByAddress", m.getBalanceByAddress)
	m.Node.HandleFunc("getWalletBalance", m.getWalletBalance)
	m.Node.HandleFunc("rescanBlockHeight", m.rescanBlockHeight)

}

//subscribe 订阅方法
func (m *MerchantNode) subscribe(ctx *owtp.Context) {

	log.Info("Merchant handle: subscribe")
	log.Info("params:", ctx.Params())

	var (
		subscriptions []*Subscription
		wallet        *openwallet.Wallet
	)

	db, err := m.OpenDB()
	if err != nil {
		responseError(ctx, err)
		return
	}
	//
	////每次订阅都先清除旧订阅
	//db.Drop("subscribe")

	for _, p := range ctx.Params().Get("subscriptions").Array() {

		s := NewSubscription(p)
		if s.WalletID == "" {
			continue
		}
		subscriptions = append(subscriptions, s)
		//log.Printf("s = %v\n", s)

		//检查是否已有钱包
		err = db.One("WalletID", s.WalletID, wallet)

		if err != nil {
			//添加订阅钱包
			wallet = openwallet.NewWallet(s.WalletID, s.Coin)
			err = db.Save(wallet)
		}

		account := wallet.SingleAssetsAccount(s.Coin)
		err = db.Save(account)
		if err != nil {
			responseError(ctx, err)
			db.Close()
			return
		}

	}

	db.Close()

	//重置订阅内容
	m.resetSubscriptions(subscriptions)

	//启动订阅交易记录任务

	responseSuccess(ctx, nil)
}

func (m *MerchantNode) createWallet(ctx *owtp.Context) {

	log.Info("Merchant handle: createWallet")
	log.Info("params:", ctx.Params())

	/*
		| 参数名称     | 类型   | 是否可空 | 描述                    |
		|--------------|--------|----------|-------------------------|
		| coin         | string | 否       | 币种标识                |
		| alias        | string | 否       | 钱包别名                |
		| passwordType | uint   | 否       | 0：自定义密码，1：协商密码 |
		| password     | string | 是       | 自定义密码              |
		| authKey    | string | 是       | 授权公钥                |
	*/

	coin := ctx.Params().Get("coin").String()
	alias := ctx.Params().Get("alias").String()
	//passwordType := ctx.Params().Get("passwordType").Uint()
	password := ctx.Params().Get("password").String()

	if len(alias) == 0 {
		responseError(ctx, errors.New("wallet alias is empty"))
		return
	}

	//导入到每个币种的数据库
	am := assets.GetMerchantAssets(coin)
	if am == nil {
		errorMsg := fmt.Sprintf("%s assets manager not found!", coin)
		responseError(ctx, errors.New(errorMsg))
		return
	}

	wallet := openwallet.Wallet{
		WalletID: openwallet.NewWalletID().String(),
		Alias:    alias,
		Password: password,
	}

	err := am.CreateMerchantWallet(&wallet)
	if err != nil {
		responseError(ctx, err)
		return
	}

	//创建单资产账户
	account := wallet.SingleAssetsAccount(coin)

	db, err := m.OpenDB()
	if err != nil {
		responseError(ctx, err)
		return
	}
	defer db.Close()

	db.Save(&wallet)
	db.Save(&account)

	log.Debug("walletID =", wallet.WalletID)

	result := map[string]interface{}{
		"coin":       coin,
		"walletID":   account.AccountID,
		"balance":    account.Balance,
		"alias":      account.Alias,
		"publicKeys": account.OwnerKeys,
	}

	responseSuccess(ctx, result)
}

func (m *MerchantNode) configWallet(ctx *owtp.Context) {

	log.Info("Merchant handle: configWallet")
	log.Info("params:", ctx.Params())

	/*

		| 参数名称 | 类型   | 是否可空 | 描述                                                           |
		|----------|--------|----------|----------------------------------------------------------------|
		| coin     | string | 否       | 币种                                                           |
		| walletID | string | 否       | 钱包ID                                                         |
		| surplus  | string | 否       | 剩余额，设置后，【余额—剩余额】低于第一笔提币金额则不提币(默认为0) |
		| fee      | string | 否       | 提币矿工费                                                     |
		| confirm  | int    | 否       | 确认次数(达到该确认次数后不再推送确认，默认30)                  |

	*/

	merchantWalletConfig := openwallet.NewWalletConfig(ctx.Params())

	if len(merchantWalletConfig.WalletID) == 0 {
		responseError(ctx, errors.New("walletID is empty"))
		return
	}

	db, err := m.OpenDB()
	if err != nil {
		responseError(ctx, err)
		return
	}
	defer db.Close()

	db.Save(merchantWalletConfig)

	responseSuccess(ctx, nil)
}

func (m *MerchantNode) getWalletList(ctx *owtp.Context) {

	log.Info("Merchant handle: getWalletList")
	log.Info("params:", ctx.Params())

	coin := ctx.Params().Get("coin").String()

	//导入到每个币种的数据库
	am := assets.GetMerchantAssets(coin)
	if am == nil {
		errorMsg := fmt.Sprintf("%s assets manager not found!", coin)
		responseError(ctx, errors.New(errorMsg))
		return
	}

	//提交给资产管理包转账
	wallets, err := m.GetMerchantWalletList()
	if err != nil {
		responseError(ctx, err)
		return
	}

	walletsMaps := make([]map[string]interface{}, 0)

	for _, w := range wallets {

		accounts, err := am.GetMerchantAssetsAccountList(w)
		if err != nil {
			continue
		}

		for _, a := range accounts {

			wmap := make(map[string]interface{})
			wmap["alias"] = a.Alias
			wmap["walletID"] = a.WalletID
			wmap["publicKeys"] = a.OwnerKeys
			wmap["coin"] = a.Symbol
			wmap["balance"] = a.Balance

			//查询钱包配置
			config, err := m.GetMerchantWalletConfig(coin, w.WalletID)
			if err == nil {
				wmap["coin"] = config.Coin
				wmap["surplus"] = config.Surplus
				wmap["fee"] = config.Fee
				wmap["confirm"] = config.Confirm
			}

			walletsMaps = append(walletsMaps, wmap)

		}
	}

	result := map[string]interface{}{
		"wallets": walletsMaps,
	}

	responseSuccess(ctx, result)
}

func (m *MerchantNode) createAddress(ctx *owtp.Context) {

	log.Info("Merchant handle: createAddress")
	log.Info("params:", ctx.Params())

	/*
		| 参数名称 | 类型   | 是否可空 | 描述         |
		|----------|--------|----------|--------------|
		| coin     | string | 否       | 币种标识     |
		| walletID | string | 否       | 钱包ID       |
		| count    | uint   | 否       | 条数         |
		| password | string | 是       | 钱包解锁密码 |
	*/

	coin := ctx.Params().Get("coin").String()
	walletID := ctx.Params().Get("walletID").String()
	count := ctx.Params().Get("count").Uint()
	password := ctx.Params().Get("password").String()

	if count == 0 {
		responseError(ctx, errors.New("create address count must be greater than 0"))
		return
	}

	//导入到每个币种的数据库
	am := assets.GetMerchantAssets(coin)
	if am == nil {
		errorMsg := fmt.Sprintf("%s assets manager not found!", coin)
		responseError(ctx, errors.New(errorMsg))
		return
	}

	//提交给资产管理包转账
	wallet, err := m.GetMerchantWalletByID(walletID)
	if err != nil {
		responseError(ctx, err)
		return
	}
	wallet.Password = password

	//导入到每个币种的数据库
	mer := assets.GetMerchantAssets(coin)
	_, err = mer.CreateMerchantAddress(wallet, wallet.SingleAssetsAccount(coin), count)

	if err != nil {
		responseError(ctx, err)
		return
	}
	/*
		addrsMaps := make([]map[string]interface{}, 0)

		for _, a := range newAddrs {
			addrsMaps = append(addrsMaps, map[string]interface{} {
				"address": a.Address,
				"walletID": a.AccountID,
				"balance":a.Balance,
				"isMemo": a.IsMemo,
				"memo": a.Memo,
				"alias": a.Alias,
			})
		}

		result := map[string]interface{}{
			"addresses": addrsMaps,
		}
	*/
	responseSuccess(ctx, nil)
}

func (m *MerchantNode) getAddressList(ctx *owtp.Context) {

	log.Info("Merchant handle: getAddressList")
	log.Info("params:", ctx.Params())

	/*
		| 参数名称  | 类型   | 是否可空 | 描述                                         |
		|-----------|--------|----------|----------------------------------------------|
		| coin      | string | 否       | 币种标识                                     |
		| walletID  | string | 否       | 钱包ID                                       |
		| watchOnly | uint   | 否       | 0: 钱包自己创建的地址,1：外部导入的订阅的地址 |
		| offset    | uint   | 是       | 从0开始                                      |
		| limit     | uint   | 是       | 查询条数                                     |
	*/

	coin := ctx.Params().Get("coin").String()
	walletID := ctx.Params().Get("walletID").String()
	watchOnly := ctx.Params().Get("watchOnly").Bool()
	offset := ctx.Params().Get("offset").Uint()
	limit := ctx.Params().Get("limit").Uint()

	//导入到每个币种的数据库
	am := assets.GetMerchantAssets(coin)
	if am == nil {
		errorMsg := fmt.Sprintf("%s assets manager not found!", coin)
		responseError(ctx, errors.New(errorMsg))
		return
	}

	//提交给资产管理包转账
	wallet, err := m.GetMerchantWalletByID(walletID)
	if err != nil {
		responseError(ctx, err)
		return
	}

	//导入到每个币种的数据库
	addrs, err := am.GetMerchantAddressList(wallet, wallet.SingleAssetsAccount(coin), watchOnly, offset, limit)

	if err != nil {
		responseError(ctx, err)
		return
	}

	addrsMaps := make([]map[string]interface{}, 0)

	for _, a := range addrs {
		addrsMaps = append(addrsMaps, map[string]interface{}{
			"address":  a.Address,
			"walletID": a.AccountID,
			"balance":  a.Balance,
			"isMemo":   common.BoolToUInt(a.IsMemo),
			"memo":     a.Memo,
			"alias":    a.Alias,
		})
	}

	result := map[string]interface{}{
		"addresses": addrsMaps,
	}

	responseSuccess(ctx, result)
}

func (m *MerchantNode) submitTransaction(ctx *owtp.Context) {

	log.Info("Merchant handle: submitTransaction")
	log.Info("params:", ctx.Params())

	var (
		//withdraws = make([]*openwallet.Withdraw, 0)
		wallets  = make(map[string][]*openwallet.Withdraw)
		tmpArray []*openwallet.Withdraw
		txIDMaps = make([]map[string]interface{}, 0)
	)

	db, err := m.OpenDB()
	if err != nil {
		responseError(ctx, err)
		return
	}

	for _, p := range ctx.Params().Get("withdraws").Map() {
		s := openwallet.NewWithdraw(p)

		if len(s.WalletID) == 0 {
			continue
		}
		var replayWith *openwallet.Withdraw
		//检查sid是否重放
		err = db.One("Sid", s.Sid, &replayWith)
		if replayWith != nil {
			//存在相关的sid不加入提现表
			errMsg := fmt.Sprintf("withdraw sid: %s is duplicate\n", s.Sid)
			//log.Printf(errMsg)

			txIDMaps = append(txIDMaps, map[string]interface{}{
				"sid":    s.Sid,
				"txid":   replayWith.TxID,
				"status": 2,
				"reason": errMsg,
			})

			continue
		}

		//withdraws = append(withdraws, s)

		tmpArray = wallets[s.WalletID]
		if tmpArray == nil {
			tmpArray = make([]*openwallet.Withdraw, 0)
		}

		tmpArray = append(tmpArray, s)
		wallets[s.WalletID] = tmpArray

	}

	db.Close()

	for wid, withs := range wallets {
		if len(withs) > 0 {
			//提交给资产管理包转账
			wallet, err := m.GetMerchantWalletByID(wid)
			if err != nil {
				responseError(ctx, err)
				return
			}
			wallet.Password = withs[0].Password
			//导入到每个币种的数据库
			mer := assets.GetMerchantAssets(withs[0].Symbol)
			if mer == nil {
				continue
			}

			walletConfig, _ := m.GetMerchantWalletConfig(withs[0].Symbol, wid)

			status := 0
			reason := ""
			txid := ""
			tx, err := mer.SubmitTransactions(wallet, wallet.SingleAssetsAccount(withs[0].Symbol), withs, walletConfig.Surplus)
			if err != nil {
				log.Error("SubmitTransactions unexpected error:", err)
				status = 3
				reason = err.Error()
				txid = ""
			} else {
				status = 1
				err = nil
				reason = ""
				txid = tx.TxID
			}

			for _, with := range withs {
				txIDMaps = append(txIDMaps, map[string]interface{}{
					"sid":    with.Sid,
					"txid":   txid,
					"status": status,
					"reason": reason,
				})

				if status == 1 {
					with.TxID = txid
					m.SaveToDB(with)
				}
			}

		}
	}

	result := map[string]interface{}{
		"withdraws": txIDMaps,
	}

	responseSuccess(ctx, result)
}

func (m *MerchantNode) getNewHeight(ctx *owtp.Context) {

	log.Info("Merchant handle: getNewHeight")
	log.Info("params:", ctx.Params())

	/*
		| 参数名称 | 类型   | 是否可空 | 描述     |
		|----------|--------|----------|----------|
		| coin     | string | 否       | 币种标识 |
	*/

	coin := ctx.Params().Get("coin").String()
	//walletID := ctx.Params().Get("walletID").String()

	am := assets.GetMerchantAssets(coin)
	blockchain, err := am.GetBlockchainInfo()
	if err != nil {
		responseError(ctx, err)
		return
	}

	result := map[string]interface{}{
		"coin":      coin,
		"cmdHeight": blockchain.ScanHeight,
		"height":    blockchain.Blocks,
	}

	responseSuccess(ctx, result)
}

func (m *MerchantNode) getWalletBalance(ctx *owtp.Context) {

	log.Info("Merchant handle: getWalletBalance")
	log.Info("params:", ctx.Params())

	/*
		| 参数名称     | 类型   | 是否可空 | 描述   |
		|--------------|--------|----------|--------|
		| coin         | string | 否       | 币名   |
		| walletID     | string | 否       | 钱包ID |
	*/

	coin := ctx.Params().Get("coin").String()
	walletID := ctx.Params().Get("walletID").String()

	am := assets.GetMerchantAssets(coin)
	balance, err := am.GetMerchantWalletBalance(walletID)
	if err != nil {
		responseError(ctx, err)
		return
	}

	result := map[string]interface{}{
		"balance": balance,
	}

	responseSuccess(ctx, result)
}

func (m *MerchantNode) getBalanceByAddress(ctx *owtp.Context) {

	log.Info("Merchant handle: getBalanceByAddress")
	log.Info("params:", ctx.Params())

	/*
		| 参数名称     | 类型   | 是否可空 | 描述   |
		|--------------|--------|----------|--------|
		| coin         | string | 否       | 币名   |
		| walletID     | string | 否       | 钱包ID |
		| address      | string  | 否        | 地址 |
	*/

	coin := ctx.Params().Get("coin").String()
	walletID := ctx.Params().Get("walletID").String()
	address := ctx.Params().Get("address").String()

	am := assets.GetMerchantAssets(coin)
	balance, err := am.GetMerchantAddressBalance(walletID, address)
	if err != nil {
		responseError(ctx, err)
		return
	}

	result := map[string]interface{}{
		"balance": balance,
	}

	responseSuccess(ctx, result)
}

func (m *MerchantNode) rescanBlockHeight(ctx *owtp.Context) {
	log.Info("Merchant handle: rescanBlockHeight")
	log.Info("params:", ctx.Params())

	/*
	| 参数名称    | 类型   | 是否可空 | 描述                   |
	|-------------|--------|----------|------------------------|
	| coin        | string | 否       | 币名                   |
	| startHeight | string | 否       | 起始高度               |
	| endHeight   | string | 否       | 结束高度，0代表最新高度 |
	*/

	var (
		err error
	)

	coin := ctx.Params().Get("coin").String()
	//walletID := ctx.Params().Get("walletID").String()
	startHeight := ctx.Params().Get("startHeight").Uint()
	endHeight := ctx.Params().Get("endHeight").Uint()

	am := assets.GetMerchantAssets(coin)
	if endHeight == 0 {
		err = am.SetMerchantRescanBlockHeight(startHeight)
	} else {
		err = am.MerchantRescanBlockHeight(startHeight, endHeight)
	}

	if err != nil {
		responseError(ctx, err)
		return
	}

	responseSuccess(ctx, nil)
}

//responseSuccess 成功结果响应
func responseSuccess(ctx *owtp.Context, result interface{}) {
	ctx.Response(result, owtp.StatusSuccess, "success")
	log.Info(ctx.Method, ":", "Response:", ctx.Resp.JsonData())
}

//responseError 失败结果响应
func responseError(ctx *owtp.Context, err error) {
	ctx.Response(nil, owtp.ErrCustomError, err.Error())
	log.Info(ctx.Method, ":", "Response:", ctx.Resp.JsonData())
}
