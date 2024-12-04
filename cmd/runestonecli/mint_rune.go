package main

import (
	"github.com/studyzy/runestone"
)

// 自转分离utxo
func test1() {
	// 自定义脚本
	btcConnector := NewMempoolConnector(config)
	prvKey, address, _ := config.GetPrivateKeyAddr()
	utxos, err := btcConnector.GetUtxos(address)
	if err != nil {
		return
	}
	tx, err := BuildSendBTCTx(prvKey, utxos, address, 50000000, config.GetFeePerByte(), config.GetNetwork())
	if err != nil {
		p.Println("BuildMintRuneTx error:", err.Error())
		return
	}
	SendTx(btcConnector, tx, nil)
}

func build_mint_rune_tx() {
	runeId, err := config.GetMint()
	if err != nil {
		p.Println(err.Error())
		return

	}
	r := runestone.Runestone{Mint: runeId}
	runeData, err := r.Encipher()
	if err != nil {
		p.Println(err)
	}
	p.Printf("Mint Rune[%s] data: 0x%x\n", config.Mint.RuneId, runeData)
	//dataString, _ := txscript.DisasmString(data)
	//p.Printf("Mint Script: %s\n", dataString)
	btcConnector := NewMempoolConnector(config)
	prvKey, address, _ := config.GetPrivateKeyAddr()
	utxos, err := btcConnector.GetUtxos(address)
	tx, err := BuildTransferBTCTx(prvKey, utxos, address, config.GetUtxoAmount(), config.GetFeePerByte(), config.GetNetwork(), runeData, false)
	if err != nil {
		p.Println("BuildMintRuneTx error:", err.Error())
		return
	}
	p.Printf("mint rune tx: %x\n", tx)

	SendTx(btcConnector, tx, nil)
}

func mint_rune_chain(num int) {
	for i := range num {
		p.Printf("%d\n", i)
		build_mint_rune_tx()
	}
}
