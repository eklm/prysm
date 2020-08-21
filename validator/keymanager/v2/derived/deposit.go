package derived

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

// SendDepositConfig contains all the required information for
// the derived keymanager to submit a 32 ETH deposit from the user's
// eth1 wallet to an eth1 RPC endpoint.
type SendDepositConfig struct {
	DepositContractAddress   string
	DepositDelaySeconds      time.Duration
	DepositPublicKeys        string
	Eth1KeystoreUTCFile      string
	Eth1KeystorePasswordFile string
	Eth1PrivateKeyFile       string
	Web3Provider             string
}

// SendDepositTx to the validator deposit contract on the eth1 chain
// using a defined configuration by first unlocking the user's eth1 wallet,
// then generating the deposit data for a desired validator account, finally
// submitting the transaction via an eth1 web3 endpoint.
func (dr *Keymanager) SendDepositTx(conf *SendDepositConfig) error {
	var txOps *bind.TransactOpts
	rpcClient, err := rpc.Dial(conf.Web3Provider)
	if err != nil {
		return err
	}
	client := ethclient.NewClient(rpcClient)
	depositAmountInGwei := params.BeaconConfig().MinDepositAmount

	if conf.Eth1PrivateKeyFile != "" {
		// User inputs private key, sign tx with private key
		privKey, err := crypto.HexToECDSA(conf.Eth1PrivateKeyFile)
		if err != nil {
			return err
		}
		txOps = bind.NewKeyedTransactor(privKey)
		txOps.Value = new(big.Int).Mul(big.NewInt(int64(depositAmountInGwei)), big.NewInt(1e9))
	} else {
		// User inputs keystore json file, sign tx with keystore json
		password := loadTextFromFile(conf.Eth1KeystorePasswordFile)

		// #nosec - Inclusion of file via variable is OK for this tool.
		keyJSON, err := ioutil.ReadFile(conf.Eth1KeystoreUTCFile)
		if err != nil {
			return err
		}
		privKey, err := keystore.DecryptKey(keyJSON, password)
		if err != nil {
			return err
		}

		txOps = bind.NewKeyedTransactor(privKey.PrivateKey)
		txOps.Value = new(big.Int).Mul(big.NewInt(int64(depositAmountInGwei)), big.NewInt(1e9))
		txOps.GasLimit = 500000
	}

	depositContract, err := contracts.NewDepositContract(common.HexToAddress(conf.DepositContractAddress), client)
	if err != nil {
		return err
	}
	keyCounter := int64(0)
	for _, validatorKey := range dr.keysCache {
		// TODO: Use a withdrawal key.
		data, depositRoot, err := depositutil.DepositInput(validatorKey, validatorKey, depositAmountInGwei)
		if err != nil {
			log.Errorf("Could not generate deposit input data: %v", err)
			continue
		}
		tx, err := depositContract.Deposit(
			txOps,
			data.PublicKey,
			data.WithdrawalCredentials,
			data.Signature,
			depositRoot,
		)
		if err != nil {
			log.Errorf("unable to send transaction to contract: %v", err)
			continue
		}

		log.WithFields(logrus.Fields{
			"Transaction Hash": fmt.Sprintf("%#x", tx.Hash()),
		}).Infof(
			"Deposit %d sent to contract address %v for validator with a public key %#x",
			keyCounter,
			conf.DepositContractAddress,
			validatorKey.PublicKey().Marshal(),
		)
		time.Sleep(conf.DepositDelaySeconds * time.Second)
		keyCounter++
	}
	return nil
}

func loadTextFromFile(filepath string) string {
	// #nosec - Inclusion of file via variable is OK for this tool.
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	scanner.Scan()
	return scanner.Text()
}
