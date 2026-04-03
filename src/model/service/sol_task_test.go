package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/gagliardetto/solana-go/rpc"
)

func TestParseSolTransaction(t *testing.T) {
	client := rpc.New("https://api.mainnet-beta.solana.com")
	sig := "3tZTwLrvmiZ59h4UzyMHPd7DPux7t9eXZgkUvEfquaoSuERrPSRNzWuSHKQM2fbiCWFDGNqoLpu2kLZnfoegVpqN"

	txInfo, err := ParseTransactionTransfers(context.Background(), client, sig)
	if err != nil {
		panic(err)
	}

	for _, item := range txInfo {
		fmt.Println("type =", item.Type)
		fmt.Println("program =", item.ProgramID)
		fmt.Println("from =", item.Source)
		fmt.Println("to =", item.Destination)
		fmt.Println("mint =", item.Mint)
		fmt.Println("authority =", item.Authority)
		fmt.Println("amount =", item.Amount)
		if item.Decimals != nil {
			fmt.Println("decimals =", *item.Decimals)
		}
		fmt.Println("-----")
	}
}

func TestFindATAAddress(t *testing.T) {
	owner := "2uFTf9TZ8gd7Kg6hkb79TxfaeNpaAgpJ8uVHguv2Yweu"
	mint := "4k3Dyjzvzp8eMZWUXbBCjEvwSkkk59S5iCNLY3QrkX6R" // ray token

	ata, err := FindATAAddress(owner, mint)
	if err != nil {
		panic(err)
	}

	fmt.Println("ATA =", ata)
}
