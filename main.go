package main

import (
    "fmt"
    "time"
    "os/exec"
    "github.com/navybluesilver/blockexplorer"
    "github.com/navybluesilver/lit-trader/trader"
  )

var (
  alice = trader.NewTrader("Alice", "127.0.0.1", 8001)
  bob = trader.NewTrader("Bob", "127.0.0.1", 8002)
  coinType = 1
)

func main() {

  // make sure we have the latest git
  update_binaries()

  // restart services
  //restartServices() //TODO: format error, do manual for now

  // Connect nodes
  connectNodes()

  // make sure that the services are running
  ping(alice)
  ping(bob)

  // wait for confirmation
  waitForFunding()

  // buy Forward Contract
  alice.Buy(1500,1)

  // wait for contract to be send to counterparty
  delaySecond(5)

  // accept all open contracts
  acceptAllOfferedContracts(bob)

  // get all contract info
  getAllContractInfo(alice)

  // get all contract info
  deleteAllContract(alice)
  deleteAllContract(bob)
}

// Services
func update_binaries() {
  runScript("./scripts/update_binaries.sh")
}

func restartServices() {
  runScript("./scripts/restart_services.sh")
}

func deleteAllContract(t *trader.Trader) {
  contracts, err := t.Lit.ListContracts()
  handleError(err)
  for _, contract := range contracts {
    fmt.Printf("[%s] - Deleting contract: %d\n", t.Name, contract.Idx)
    t.Lit.DeleteContract(contract.Idx)
  }
}

func runScript(script string) {
  cmd := exec.Command(script)
  stdout, err := cmd.Output()

  if err != nil {
      println(err.Error())
      return
  }
  print(string(stdout))
}

// Tasks
func ping(t *trader.Trader) {
  isListening, err := t.Lit.IsListening()
  handleError(err)


  if isListening {
    log(fmt.Sprintf("[%s] - Listening", t.Name))
  } else {
    log(fmt.Sprintf("[%s] - Not listening\n", t.Name))
  }
}

func hasFunding(t *trader.Trader) (hasFunding bool) {

  witnessAddress := getWitnessAddress(t)
  legacyAddress := getLegacyAddress(t)


  // confirmed witness funding, return true
  if hasBalance(t, true) {
    return true
  }

  // confirmed legacy funding, sweep Witness Addres, return false and wait
  if blockexplorer.HasBalance(legacyAddress,true) {
        if !checkBlockHeight(t) {
          return false
        }
        log(fmt.Sprintf("Legacy Address has confirmed funding, now sweeping to the Witness Address: %s", witnessAddress))
        sweepFunds(t)
        return false
  }

  // unconfirmed legacy funding, return false and wait
  if blockexplorer.HasBalance(legacyAddress,false) {
          log(fmt.Sprintf("Legacy Address has been funded, but still waiting for confirmations on the blockchain: %s", legacyAddress))
          return false
  }


  log(fmt.Sprintf("Legacy Address needs funding: %s", legacyAddress))
  return false
}

func sweepFunds(t *trader.Trader) {
  t.Lit.Sweep(getWitnessAddress(t),1)
}

func getLegacyAddress(t *trader.Trader) (pubKey string) {
  addr, err := t.Lit.GetAddresses(uint32(coinType), 0, true)
  handleError(err)
  return addr[0]
}

func getWitnessAddress(t *trader.Trader) (pubKey string) {
  addr, err := t.Lit.GetAddresses(uint32(coinType), 0, false)
  handleError(err)
  return addr[0]
}

func getBalance(t *trader.Trader, witness bool) (int) {
  bal, err := t.Lit.ListBalances()
  handleError(err)

  utxo := 0
  matureWitty := 0
  for _, b := range bal {
      fmt.Printf("[%s] - Channel: %d | UTXO: %d | Confirmed Witness: %d \n", t.Name, b.ChanTotal, b.TxoTotal, b.MatureWitty)
      utxo = utxo + int(b.TxoTotal)
      matureWitty = matureWitty + int(b.MatureWitty)
  }

  if witness {
      return matureWitty
  } else {
    return utxo
  }
}

func hasBalance(t *trader.Trader, witness bool) (bool) {
  bal := getBalance(t, witness)
  if bal > 0 {
      return true
  } else {
    return false
  }
}

func checkBlockHeight(t *trader.Trader) (ok bool) {
  litHeight := getBlockHeight(t)
  testnetHeight := blockexplorer.GetBlockHeight()
  delta := testnetHeight - litHeight
  if delta < 0 {
    return true
  }

  if delta > 0 {
    log(fmt.Sprintf("Block height for %s is only [%d], while expecting [%d]", t.Name, litHeight, testnetHeight))
    return false
  }

  return true
}

func getBlockHeight(t *trader.Trader) (int) {
  bal, err := t.Lit.ListBalances()
  handleError(err)
  return int(bal[0].SyncHeight)
}

func acceptAllOfferedContracts(t *trader.Trader) {
  contracts, err := t.Lit.ListContracts()
  handleError(err)
  for _, contract := range contracts {
      if contract.Status == 2 {
          fmt.Printf("Accepting contract [%d]\n", contract.Idx)
          fmt.Printf("OurFundingInputs count: %d\n", len(contract.OurFundingInputs))
          fmt.Printf("TheirFundingInputs count: %d\n", len(contract.TheirFundingInputs))
          fmt.Printf("TheirSettlementSignatures count: %d\n", len(contract.TheirSettlementSignatures))
          fmt.Printf("Division count: %d\n", len(contract.Division))
          t.Lit.AcceptContract(contract.Idx)
      }
  }
}

func getAllContractInfo(t *trader.Trader) {
  contracts, err := t.Lit.ListContracts()
  handleError(err)
  for _, contract := range contracts {
          fmt.Printf("contract [%d]\n", contract.Idx)
          fmt.Printf("OurFundingInputs count: %d\n", len(contract.OurFundingInputs))
          fmt.Printf("TheirFundingInputs count: %d\n", len(contract.TheirFundingInputs))
          fmt.Printf("TheirSettlementSignatures count: %d\n", len(contract.TheirSettlementSignatures))
          fmt.Printf("Division count: %d\n", len(contract.Division))
          fmt.Println("")
  }
}

// Waits
func delaySecond(n time.Duration) {
    log("waiting...")
    time.Sleep(n * time.Second)
}

func waitForFunding() {
    for wait := !bothHasFunding(); wait; wait = !bothHasFunding() {
      delaySecond(60)
    }
}

func bothHasFunding() (bool) {
  fundingAlice := hasFunding(alice)
  fundingBob := hasFunding(bob)
  if fundingAlice && fundingBob {
    return true
  }
  return false
}

// Error Handling
func handleError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func log(message string) {
  fmt.Println(message)
}

func connectNodes() {
	// Instruct both nodes to listen for incoming connections
	err := alice.Lit.Listen(":2448")
	handleError(err)
	err = bob.Lit.Listen(":2449")
	handleError(err)

  // Connect Alice and Bob
  log(fmt.Sprintf("Connecting %s and %s", alice.Name, bob.Name))
	lnAdr, err := bob.Lit.GetLNAddress()
	handleError(err)
	alice.Lit.Connect(lnAdr, bob.Host, 2449)
}
