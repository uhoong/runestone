package main

import (
	"bytes"
	"errors"
	"log"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
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

// 指定要使用的utxo铸造符文，包括发送过程
// num：打符文比数
func mint_rune(num int64, tx_id string, index int) {
	btcConnector := NewMempoolConnector(config)
	tx_chain, err := build_mint_rune_chain(num, tx_id, index)
	if err != nil {
		p.Print(err.Error())
		return
	}
	log.Println("开始发送交易")
	for i, tx := range tx_chain {
		log.Printf("交易笔数%d", i)
		_, err := btcConnector.SendRawTransaction(tx, false)
		if err != nil {
			p.Printf(err.Error())
			return
		}
	}
}

// 指定要使用的utxo构造铭文交易链
// num：打符文比数
func build_mint_rune_chain(num int64, tx_id string, index int) ([]*wire.MsgTx, error) {
	btcConnector := NewMempoolConnector(config)
	runeId, err := config.GetMint()
	if err != nil {
		p.Println(err.Error())
		return nil, err
	}
	utxo, err := btcConnector.get_utxo_by_hash_index(tx_id, index)
	if err != nil {
		return nil, err
	}
	initial_value := btcutil.Amount(utxo.Value)
	recommend_fee, err := btcConnector.get_recommended_fee()
	if err != nil {
		return nil, err
	}
	// 根据给定的utxo，构造一个交易链
	tx_chain := make([]*wire.MsgTx, 0)

	for i := 0; i < int(num); i++ {
		tx := build_mint_rune_tx(runeId, utxo)
		tx_chain = append(tx_chain, tx)
		tx_out := tx.TxOut[1]
		tx_hash := tx.TxHash()
		utxo = &Utxo{
			Index:    1,
			Value:    tx_out.Value,
			PkScript: tx_out.PkScript,
			TxHash:   BytesToHash(tx_hash.CloneBytes()),
		}
		p.Printf("mint rune tx hash: %s\n", tx_hash)
	}
	single_tx_vsize := mempool.GetTxVirtualSize(btcutil.NewTx(tx_chain[len(tx_chain)-1]))
	all_fee := btcutil.Amount(single_tx_vsize * config.FeePerByte * num)
	if initial_value-all_fee < btcutil.Amount((recommend_fee*1.5-float64(config.FeePerByte))*float64(single_tx_vsize*num)) {
		return nil, errors.New("insufficient balance")
	}
	return tx_chain, nil
}

// 使用给定的utxo构建tx
func build_mint_rune_tx(runeId *runestone.RuneId, utxo *Utxo) *wire.MsgTx {
	r := runestone.Runestone{Mint: runeId}
	runeData, err := r.Encipher()
	if err != nil {
		p.Println(err)
	}
	p.Printf("Mint Rune[%s] data: 0x%x\n", config.Mint.RuneId, runeData)
	//dataString, _ := txscript.DisasmString(data)
	//p.Printf("Mint Script: %s\n", dataString)
	// btcConnector := NewMempoolConnector(config)
	prvKey, address, _ := config.GetPrivateKeyAddr()

	tx, err := build_transfer_btc_tx(prvKey, utxo, address, config.GetUtxoAmount(), config.GetFeePerByte(), config.GetNetwork(), runeData, false)

	if err != nil {
		p.Println("BuildMintRuneTx error:", err.Error())
		return nil
	}
	return tx
}

func build_transfer_btc_tx(privateKey *btcec.PrivateKey, utxo *Utxo, toAddr string, toAmount, feeRate int64, net *chaincfg.Params, runeData []byte, splitChangeOutput bool) (*wire.MsgTx, error) {
	address, err := btcutil.DecodeAddress(toAddr, net)
	if err != nil {
		return nil, err
	}
	pkScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return nil, err
	}
	// 1. build tx
	transfer_tx, err := build_rune_tx(utxo, wire.NewTxOut(toAmount, pkScript), feeRate, runeData, splitChangeOutput)
	if err != nil {
		return nil, err
	}
	// 2.sign tx
	transfer_tx, err = sign_rune_tx(privateKey, utxo, transfer_tx)
	if err != nil {
		return nil, err
	}
	return transfer_tx, nil
	// // 3. serialize
	// commitTxBytes, err := serializeTx(transfer_tx)
	// if err != nil {
	// 	return nil, err
	// }
	// return commitTxBytes, nil
}

func build_rune_tx(commitTxOutPoint *Utxo, revealTxPrevOutput *wire.TxOut, commitFeeRate int64, runeData []byte, splitChangeOutput bool) (*wire.MsgTx, error) {
	totalSenderAmount := btcutil.Amount(0)
	totalRevealPrevOutput := revealTxPrevOutput.Value
	tx := wire.NewMsgTx(wire.TxVersion)
	var changePkScript *[]byte

	txOut := commitTxOutPoint.TxOut()
	outPoint := commitTxOutPoint.OutPoint()
	if changePkScript == nil { // first sender as change address
		changePkScript = &txOut.PkScript
	}
	in := wire.NewTxIn(&outPoint, nil, nil)
	in.Sequence = defaultSequenceNum
	tx.AddTxIn(in)

	totalSenderAmount += btcutil.Amount(txOut.Value)
	tx.AddTxOut(wire.NewTxOut(0, runeData))
	// add reveal tx output
	tx.AddTxOut(revealTxPrevOutput)
	if splitChangeOutput || !bytes.Equal(*changePkScript, revealTxPrevOutput.PkScript) {
		// add change output
		tx.AddTxOut(wire.NewTxOut(0, *changePkScript))
	}
	//mock witness to calculate fee
	emptySignature := make([]byte, 64)
	for _, in := range tx.TxIn {
		in.Witness = wire.TxWitness{emptySignature}
	}
	fee := btcutil.Amount(mempool.GetTxVirtualSize(btcutil.NewTx(tx))) * btcutil.Amount(commitFeeRate)
	changeAmount := totalSenderAmount - btcutil.Amount(totalRevealPrevOutput) - fee
	if changeAmount > 0 {
		tx.TxOut[len(tx.TxOut)-1].Value += int64(changeAmount)
	} else {
		return nil, errors.New("insufficient balance")
	}
	//clear mock witness
	for _, in := range tx.TxIn {
		in.Witness = nil
	}
	return tx, nil
}

// 构造rbf加速交易
// tx_id为打符文的最后一笔交易id
// tx_num为交易笔数
// increase为增加的交易费
func build_rbf_tx(tx_id string, tx_num int64, increase_fee float64) (*wire.MsgTx, error) {
	btc_connector := NewMempoolConnector(config)
	tx, err := btc_connector.GetRawTxByHash(tx_id)
	if err != nil {
		return nil, err
	}
	tx_size := mempool.GetTxVirtualSize(btcutil.NewTx(tx))
	tx_out := tx.TxOut[1]
	tx_in := tx.TxIn[0]
	if tx_out.Value < int64(increase_fee*float64(tx_num*tx_size))+1 {
		return nil, errors.New("rbf insufficent balance")
	}
	tx_out.Value -= int64(increase_fee*float64(tx_num*tx_size)) + 1
	// 获取上一笔交易的utxo进行签名
	utxo, err := btc_connector.get_utxo_by_hash_index(tx_in.PreviousOutPoint.Hash.String(), int(tx_in.PreviousOutPoint.Index))
	if err != nil {
		return nil, err
	}
	prvKey, _, _ := config.GetPrivateKeyAddr()
	tx, err = sign_rune_tx(prvKey, utxo, tx)
	if err != nil {
		return nil, err
	}
	_, err = btc_connector.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func sign_rune_tx(prvKey *btcec.PrivateKey, utxo *Utxo, commitTx *wire.MsgTx) (*wire.MsgTx, error) {
	tx_out := utxo.TxOut()
	tx_in := commitTx.TxIn[0]
	witness, err := txscript.TaprootWitnessSignature(commitTx, txscript.NewTxSigHashes(commitTx, utxo),
		0, tx_out.Value, tx_out.PkScript, txscript.SigHashDefault, prvKey)
	if err != nil {
		return nil, err
	}
	tx_in.Witness = witness
	return commitTx, nil
}
