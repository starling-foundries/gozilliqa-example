package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/Zilliqa/gozilliqa-sdk/account"
	"github.com/Zilliqa/gozilliqa-sdk/bech32"
	"github.com/Zilliqa/gozilliqa-sdk/contract"
	"github.com/Zilliqa/gozilliqa-sdk/keytools"
	"github.com/Zilliqa/gozilliqa-sdk/provider"
	"github.com/Zilliqa/gozilliqa-sdk/transaction"
	"github.com/Zilliqa/gozilliqa-sdk/util"
)

func testBlockchain() {
	zilliqa := provider.NewProvider("https://dev-api.zilliqa.com/")

	// These are set by the core protocol, and may vary per-chain.
	// You can manually pack the bytes according to chain id and msg version.
	// For more information: https://apidocs.zilliqa.com/?shell#getnetworkid

	const chainID = 333  // chainId of the developer testnet
	const msgVersion = 1 // current msgVersion
	VERSION := util.Pack(chainID, msgVersion)

	// Populate the wallet with an account
	const privateKey = "3375F915F3F9AE35E6B301B7670F53AD1A5BE15D8221EC7FD5E503F21D3450C8"

	user := account.NewWallet()
	user.AddByPrivateKey(privateKey)
	user.SetDefault("8254b2c9acdf181d5d6796d63320fbb20d4edd12")
	addr := keytools.GetAddressFromPrivateKey(util.DecodeHex(privateKey))
	fmt.Println("My account address is:", user.DefaultAccount.Address)
	fmt.Println("Converting from private key gives:", addr)
	bech, _ := bech32.ToBech32Address(user.DefaultAccount.Address)
	fmt.Println("The bech32 address is:", bech)

	//testing Transaction methods
	bal, _ := zilliqa.GetBalance(user.DefaultAccount.Address).Result.(map[string]interface{})["balance"]
	gas := zilliqa.GetMinimumGasPrice().Result

	fmt.Println("The balance for account ", user.DefaultAccount.Address, " is: ", bal)
	fmt.Println("The blockchain reports minimum gas price: ", gas)

	fmt.Println("Constructing a transfer transaction...")

	//TODO: parse rpc response to use gas variable for gas price
	tx := &transaction.Transaction{
		Version:      strconv.FormatInt(int64(VERSION), 10),
		SenderPubKey: string(user.DefaultAccount.PublicKey),
		ToAddr:       "A54E49719267E8312510D7b78598ceF16ff127CE",
		Amount:       "10000000",
		GasPrice:     "1000000000",
		GasLimit:     "1",
		Code:         "",
		Data:         "",
		Priority:     false,
	}

	//sign transaction
	fmt.Println("Trying to sign the transaction")

	err := user.SignWith(tx, "8254B2C9ACDF181D5D6796D63320FBB20D4EDD12", *zilliqa)
	if err != nil {
		fmt.Println("Signing transaction, error thrown:", err)
		fmt.Printf("your wallet looks like this: %v", *user)
	}

	// Send a transaction to the network
	// fmt.Println("Sending a payment transaction to the network...")
	// rsp := zilliqa.CreateTransaction(tx.ToTransactionPayload())
	// if rsp.Error != nil {
	// 	fmt.Println("Transaction response error recieved: ", rsp.Error)
	// } else {
	// 	result := rsp.Result.(map[string]interface{})
	// 	hash := result["TranID"].(string)
	// 	fmt.Printf("sent transaction hash is %s", hash)
	// 	tx.Confirm(hash, 1000, 3, zilliqa)
	// }

	init := []contract.Value{
		{
			"_scilla_version",
			"Uint32",
			"0",
		},
		{
			"owner",
			"ByStr20",
			"0x8254b2c9acdf181d5d6796d63320fbb20d4edd12",
		},
	}
	code, _ := ioutil.ReadFile("./HelloWorld.scilla")

	fmt.Println("Attempting to deploy Hello World smart contract...")

	hello := contract.Contract{
		Code:     string(code),
		Init:     init,
		Singer:   user,
		Provider: zilliqa,
	}
	nonce, err := zilliqa.GetBalance(string(user.DefaultAccount.Address)).Result.(map[string]interface{})["nonce"].(json.Number).Int64()
	if err != nil {
		fmt.Println("Nonce response error thrown: ", err)
	}
	deployParams := contract.DeployParams{
		Version:      strconv.FormatInt(int64(VERSION), 10),
		Nonce:        strconv.FormatInt(nonce+1, 10),
		GasPrice:     "10000000000",
		GasLimit:     "10000",
		SenderPubKey: string(user.DefaultAccount.PublicKey),
	}
	deployTx, err := DeployWith(&hello, deployParams, "8254B2C9ACDF181D5D6796D63320FBB20D4EDD12")

	if err != nil {
		fmt.Println("Contract deployment failed with error: ", err)
	}

	deployTx.Confirm(deployTx.ID, 1000, 10, zilliqa)

	//verify that the contract is deployed

}

func DeployWith(c *contract.Contract, params contract.DeployParams, pubkey string) (*transaction.Transaction, error) {
	if c.Code == "" || c.Init == nil || len(c.Init) == 0 {
		return nil, errors.New("Cannot deploy without code or initialisation parameters.")
	}

	tx := &transaction.Transaction{
		ID:           params.ID,
		Version:      params.Version,
		Nonce:        params.Nonce,
		Amount:       "0",
		GasPrice:     params.GasPrice,
		GasLimit:     params.GasLimit,
		Signature:    "",
		Receipt:      transaction.TransactionReceipt{},
		SenderPubKey: params.SenderPubKey,
		ToAddr:       "0000000000000000000000000000000000000000",
		Code:         strings.ReplaceAll(c.Code, "/\\", ""),
		Data:         c.Init,
		Status:       0,
	}

	err2 := c.Singer.SignWith(tx, pubkey, *c.Provider)
	if err2 != nil {
		return nil, err2
	}

	rsp := c.Provider.CreateTransaction(tx.ToTransactionPayload())

	if rsp.Error != nil {
		return nil, errors.New(rsp.Error.Message)
	}

	result := rsp.Result.(map[string]interface{})
	hash := result["TranID"].(string)
	contractAddress := result["ContractAddress"].(string)

	tx.ID = hash
	tx.ContractAddress = contractAddress
	return tx, nil

}

func main() {
	testBlockchain()
}
