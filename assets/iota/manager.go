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

package iota

import (
	"path/filepath"


	//"time"
	"github.com/astaxie/beego/config"
	"github.com/shopspring/decimal"
	"errors"
	"github.com/iotaledger/giota"
	"fmt"
	"log"
	"time"
	"github.com/blocktree/OpenWallet/common"
	"github.com/blocktree/OpenWallet/common/file"
)

//loadConfig 读取配置
func loadConfig() error {

	var (
		c   config.Configer
		err error
	)

	//读取配置
	absFile := filepath.Join(configFilePath, configFileName)
	c, err = config.NewConfig("ini", absFile)
	if err != nil {
		return errors.New("Config is not setup. Please run 'wmd config -s <symbol>' ")
	}

	serverAPI = c.String("apiURL")
	threshold, _ = decimal.NewFromString(c.String("threshold"))
	sumAddress = c.String("sumAddress")
	rpcUser = c.String("rpcUser")
	rpcPassword = c.String("rpcPassword")
	nodeInstallPath = c.String("nodeInstallPath")
	isTestNet, _ = c.Bool("isTestNet")
	if isTestNet {
		walletDataPath = c.String("testNetDataPath")
	} else {
		walletDataPath = c.String("mainNetDataPath")
	}


	//token := basicAuth(rpcUser, rpcPassword)

	//client = &Client{
	//	BaseURL:     serverAPI,
	//	Debug:       false,
	//	AccessToken: token,
	//}

	return nil
}

/*
//BackupWallet 备份数据
func BackupWallet(name string,seed giota.Trytes) (string, error) {

	//创建备份文件夹
	newBackupDir := filepath.Join(backupDir, name + "-backup-"+common.TimeFormat("20060102150405"))
	file.MkdirAll(newBackupDir)

	//创建临时备份文件wallet.dat
	tmpWalletDat := fmt.Sprintf("tmp-walllet-%d.dat", time.Now().Unix())
	tmpWalletDat = filepath.Join(walletDataPath, tmpWalletDat)

	////1. 备份核心钱包的wallet.dat
	//err = BackupWalletData(tmpWalletDat)
	//if err != nil {
	//	return "", err
	//}

	//复制临时文件到备份文件夹
	file.Copy(tmpWalletDat, filepath.Join(newBackupDir, "wallet.dat"))

	//删除临时文件
	file.Delete(tmpWalletDat)

	//2. 备份种子文件
	file.Copy(filepath.Join(keyDir, name+".key"), newBackupDir)

	//3. 备份地址数据库
	file.Copy(filepath.Join(dbPath, name+".db"), newBackupDir)

	return newBackupDir, nil
}
*/

// NewAddresses generates new count addresses from seed with a checksum
func NewAddressesWithChecksum(seed giota.Trytes, start, count, security int) ([]giota.Address, error) {
	as := make([]giota.Address, count)

	for i := 0; i < count; i++ {
		adrWithoutChecksum, err := giota.NewAddress(seed, start+i, security)
		if err != nil {
			return nil, err
		}
		as[i] = giota.Address(adrWithoutChecksum.WithChecksum())
	}
	return as, nil
}


func CreateAddresses(seed giota.Trytes, start, count, security int) (string,error ){

	var(
		adrsNote1 string
	)
	adrs,err:=giota.NewAddresses(seed,start,count,security) //without checksum.
	if err != nil {
		return "",err
	}else {
		//t.Logf("start from %d end in %d, security level: %d\nTestNewAddresses() = %#v\n", start, start+count-1, security, adr)
		log.Printf("Addresses without Checksum, start from %d end in %d, security level: %d\n", start, start+count-1, security)
		adrsNote1 = fmt.Sprintf("Addresses without Checksum, start from %d end in %d, security level: %d\n", start, start+count-1, security)
		for i := 0; i < count; i++ {
			log.Printf("%s\n",string(adrs[i]))
		}
	}
	//log.Printf("\n")
	//adrsWithChecksum,err:=NewAddressesWithChecksum(seed,start,count,security) //without checksum.
	//if err != nil {
	//	fmt.Errorf("TestNewAddresses([]) expected err to be nil but got %v", err)
	//}else {
	//	//t.Logf("start from %d end in %d, security level: %d\nTestNewAddresses() = %#v\n", start, start+count-1, security, adr)
	//	log.Printf("Addresses with Checksum, start from %d end in %d, security level: %d\n", start, start+count-1, security)
	//	for i := 0; i < count; i++ {
	//		log.Printf("%s\n",string(adrsWithChecksum[i]))
	//	}
	//}

	timestamp := time.Now()
	//建立文件名，时间格式2006-01-02 15:04:05
	filename := "address-" + common.TimeFormat("20060102150405", timestamp) + ".txt"
	filePath := filepath.Join(addressDir, filename)

	var (
		content string
		//contentWithoutChecksum string
	)
	for i:=0;i< len(adrs);i++ {
		content = content + string(adrs[i]) + "\n"
	}
	file.MkdirAll(addressDir)
	file.WriteFile(filePath, []byte(adrsNote1),true)
	file.WriteFile(filePath, []byte(content), true)

	return filePath,nil
}

func GetWalletInfo(seed string)(giota.Address,[]giota.Address,int64,error){
	var (
		err  error
		adr  giota.Address
		adrs []giota.Address
	)

	trytes,err:=giota.ToTrytes(seed)
	if err != nil{
		return "",nil,0,err
	}

	for i := 0; i < 5; i++ {
		api := giota.NewAPI(giota.RandomNode(), nil)
		adr, adrs, err = giota.GetUsedAddress(api, trytes, 2)
		if err == nil {
			break
		}
	}

	if err != nil {
		return "",nil,0,err
	}

	//t.Log(adr, adrs)
	if len(adrs) < 1 {
		fmt.Errorf("GetUsedAddress is incorrect")
		return "",nil,0,nil
	}

	//add by chenzhiwen
	var totalBalance int64
	for i:=0;i< len(adrs);i++{
		api := giota.NewAPI(giota.RandomNode(), nil)
		resp, err := api.GetBalances([]giota.Address{adrs[i]}, 100)
		if err == nil {
			totalBalance += resp.Balances[0]
		}
	}
	//fmt.Printf("Total Balance = %d\n",totalBalance)
	return adr,adrs,totalBalance,nil
}

func SendTransaction(seed string,address giota.Address,value int64,tag giota.Trytes) error{

	var err error
	trytes,err:=giota.ToTrytes(seed)
	if err != nil{
		return err
	}

	trs := []giota.Transfer{
		giota.Transfer{
			Address: address,
			Value:   value,
			Tag:     tag,
		},
	}

	var bdl giota.Bundle
	for i := 0; i < 5; i++ {
		api := giota.NewAPI(giota.RandomNode(), nil)
		bdl, err = giota.PrepareTransfers(api, trytes, trs, nil, "", 2)
		if err == nil {
			break
		}
	}

	if err != nil {
		return err
	}

	if len(bdl) < 3 {
		for _, tx := range bdl {
			log.Printf(string(tx.Trytes()))
		}
		fmt.Errorf("PrepareTransfers is incorrect len(bdl)=%d", len(bdl))
		return nil
	}

	if err = bdl.IsValid(); err != nil {
		return err
	}

	name, pow := giota.GetBestPoW()
	log.Printf("using PoW: %s", name)

	for i := 0; i < 5; i++ {
		api := giota.NewAPI(giota.RandomNode(), nil)
		bdl, err = giota.Send(api, trytes, 2, trs, 18, pow)
		if err == nil {
			break
		}
	}

	if err != nil {
		return err
	}

	for _, tx := range bdl {
		log.Printf(string(tx.Trytes()))
	}

	return nil
}