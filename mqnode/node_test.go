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

package mqnode

import (
	"path/filepath"
	"github.com/blocktree/OpenWallet/owtp"
	"testing"
	"time"
	"github.com/streadway/amqp"
	"github.com/blocktree/OpenWallet/manager"
)
var mqURL = "192.168.30.160:5672"
var nodeConfig NodeConfig
func init() {
	nodeConfig = NodeConfig{
		MerchantNodeURL: mqURL,
		CacheFile:       filepath.Join(merchantDir, cacheFile),
		ConnectType:owtp.MQ,
		Exchange:"DEFAULT_EXCHANGE",
		QueueName:"ALL_READ",
		ReceiveQueueName:"ALL_WRITE",
		Account:"admin",
		Password:"admin",
	}
}


func TestNewNode(t *testing.T) {

	nodeConfig := NodeConfig{
		MerchantNodeURL :mqURL,
		ConnectType     :"mq",
		Exchange     :  "DEFAULT_EXCHANGE",
		QueueName     :  "Test",
		ReceiveQueueName  :     "OW_RPC_JAVA",
		Account    :   "admin",
		Password    :   "admin",
	}
	node,_ := NewBitNodeNode(nodeConfig)
	config := manager.NewConfig()
	node.manager = manager.NewWalletManager(config)
	node.manager.Init()
	node.Run()
	time.Sleep(10000 * time.Second)


}


const (
	exchange = "DEFAULT_EXCHANGE"
	queueName = "OW_RPC_JAVA"
	//mqurl ="amqp://aielves:aielves12301230@39.108.64.191:36672/"

	mqurl ="amqp://admin:admin@192.168.30.160:5672/"
)


func TestCreateWallet(t *testing.T) {
	var conn *amqp.Connection
	var channel *amqp.Channel
	json := `{"d":{"authKey":"e10adc3949ba59abbe56e057f20f883e","password":"123456","appID":"b4b1962d415d4d30ec71b28769fda585","alias":"我的钱包","passwordType":1,"isTrust":1},"m":"createWallet","n":"239424950006583296","r":1,"t":1537257022}`
	conn, _ = amqp.Dial(mqurl)
	channel, _ = conn.Channel()


	channel.Publish(exchange, queueName, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(json),
	})
	time.Sleep(1*time.Second)
	//nodeConfig := NodeConfig{
	//	MerchantNodeURL :mqURL,
	//	ConnectType     :"mq",
	//	Exchange     :  "DEFAULT_EXCHANGE",
	//	QueueName     :  "Test",
	//	ReceiveQueueName  :   "OW_RPC_JAVA",
	//	Account    :   "admin",
	//	Password    :   "admin",
	//}
	//node,_ := NewBitNodeNode(nodeConfig)
	//config := manager.NewConfig()
	//node.manager = manager.NewWalletManager(config)
	//node.manager.Init()
	//node.Run()
	//time.Sleep(10000 * time.Second)
}

func TestCreateAddress(t *testing.T) {
	var conn *amqp.Connection
	var channel *amqp.Channel
	json := `{"r":1,"t":1537267858,"d":{"walletID":"WDdipmoXN9zRG9r4m1SndeXfqtJqXckn9W","accountID":"L68oA5FdEFxkT17wacymBcjbmNSbcGYE2c43ZCkC1UPrqpJ6qR","appID":"b4b1962d415d4d30ec71b28769fda585","count":2},"m":"createAddress","n":"239470400399085568"}`
	conn, _ = amqp.Dial(mqurl)
	channel, _ = conn.Channel()


	channel.Publish(exchange, queueName, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(json),
	})
	//nodeConfig := NodeConfig{
	//	MerchantNodeURL :mqURL,
	//	ConnectType     :"mq",
	//	Exchange     :  "DEFAULT_EXCHANGE",
	//	QueueName     :  "Test",
	//	ReceiveQueueName  :   "OW_RPC_JAVA",
	//	Account    :   "admin",
	//	Password    :   "admin",
	//}
	//node,_ := NewBitNodeNode(nodeConfig)
	//config := manager.NewConfig()
	//node.manager = manager.NewWalletManager(config)
	//node.manager.Init()
	//node.Run()
	//time.Sleep(10000 * time.Second)
}

func TestCreateAssetsAccount(t *testing.T) {
	var conn *amqp.Connection
	var channel *amqp.Channel
	json := `{"d":{"walletID":"WDdipmoXN9zRG9r4m1SndeXfqtJqXckn9W","symbol":"BTC","password":"123456","appID":"b4b1962d415d4d30ec71b28769fda585","alias":"我的资产账户","otherOwnerKeys":"","isTrust":1,"reqSigs":1},"m":"createAssetsAccount","n":"239379936731860992","r":1,"t":1537246290}`
	conn, _ = amqp.Dial(mqurl)
	channel, _ = conn.Channel()


	channel.Publish(exchange, queueName, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(json),
	})
	//nodeConfig := NodeConfig{
	//	MerchantNodeURL :mqURL,
	//	ConnectType     :"mq",
	//	Exchange     :  "DEFAULT_EXCHANGE",
	//	QueueName     :  "Test",
	//	ReceiveQueueName  :     "OW_RPC_JAVA",
	//	Account    :   "admin",
	//	Password    :   "admin",
	//}
	//node,_ := NewBitNodeNode(nodeConfig)
	//config := manager.NewConfig()
	//node.manager = manager.NewWalletManager(config)
	//node.manager.Init()
	//node.Run()
	//time.Sleep(10000 * time.Second)
}

func TestCreateTransaction(t *testing.T) {
	var conn *amqp.Connection
	var channel *amqp.Channel
	json := `{"d":{"accountID":"KcYEWNt8T8xYfZBPyxs5MdGsKbYRuoUyqNzfqPkLGxjjbdZEvH","amount":"1.08","address":"mgU7H36xabdHWi9RHKvTJu3Nfd1hNTFQhQ","appID":"b4b1962d415d4d30ec71b28769fda585","memo":"","feeRate":"0.001","coin":{"isContract":1,"symbol":"BTC","contractID":""},"sid":"239381340338917376"},"m":"createTransaction","n":"239381351374131200","r":1,"t":1537246627}`
	conn, _ = amqp.Dial(mqurl)
	channel, _ = conn.Channel()


	channel.Publish(exchange, queueName, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(json),
	})
	//nodeConfig := NodeConfig{
	//	MerchantNodeURL :mqURL,
	//	ConnectType     :"mq",
	//	Exchange     :  "DEFAULT_EXCHANGE",
	//	QueueName     :  "Test",
	//	ReceiveQueueName  :     "OW_RPC_JAVA",
	//	Account    :   "admin",
	//	Password    :   "admin",
	//}
	//node,_ := NewBitNodeNode(nodeConfig)
	//config := manager.NewConfig()
	//node.manager = manager.NewWalletManager(config)
	//node.manager.Init()
	//node.Run()
	//time.Sleep(10000 * time.Second)
}