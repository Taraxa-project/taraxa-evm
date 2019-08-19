package taraxa_vm

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/taraxa/taraxa_types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/metric_utils"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"golang.org/x/text/secure/precis"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/debug"
	"testing"
)

func TestFoo(t *testing.T) {


}

func BenchmarkStateTransitionTestMode(b *testing.B) {
	var cfg = new(struct {
		*VmConfig                     `json:"vmConfig"`
		*types.StateTransitionRequest `json:"stateTransitionRequest"`
	})
	config_file_path := os.Getenv("CONFIG_FILE")
	bytes, err := ioutil.ReadFile(config_file_path)
	util.PanicIfPresent(err)
	util.PanicIfPresent(json.Unmarshal(bytes, cfg))
	taraxaVM, _, createVmErr := cfg.VmConfig.NewVM()
	util.PanicIfPresent(createVmErr)
	allTransactions := taraxa_types.NewTxIdSet(nil)
	for txId := range cfg.Block.Transactions {
		allTransactions.Add(txId)
	}
	concurrentSchedule, _, scheduleErr := taraxaVM.GenerateSchedule(cfg.StateTransitionRequest)
	util.PanicIfPresent(scheduleErr)
	fmt.Println("tx count", len(cfg.Block.Transactions))
	fmt.Println("confict %:",
		float64(concurrentSchedule.SequentialTransactions.Size())/float64(len(cfg.Block.Transactions)))
	debug.SetGCPercent(-1)

	for i := 0; i < 10; i++ {
		rec := metric_utils.NewTimeRecorder()
		taraxaVM.TestMode(cfg.StateTransitionRequest, &TestModeParams{})
		fmt.Println("elapsed", rec())

	}
	for i := 0; i < 10; i++ {
		rec := metric_utils.NewTimeRecorder()
		taraxaVM.TestMode(cfg.StateTransitionRequest, &TestModeParams{})
		fmt.Println("elapsed", rec())
	}
	runtime.GC()

	benchmark := func(params *TestModeParams) func(b *testing.B) {
		return func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				taraxaVM.TestMode(cfg.StateTransitionRequest, params)
				b.StopTimer()
				runtime.GC()
				b.StartTimer()
			}
		}
	}
	//b.Run("read_write_parallel_separate_db", benchmark(&TestModeParams{
	//	DoCommitsInSeparateDB: true,
	//}))
	//b.Run("read_write_parallel", benchmark(&TestModeParams{
	//	DoCommits: true,
	//}))
	//b.Run("read_write_parallel_commit_sync", benchmark(&TestModeParams{
	//	CommitSync: true,
	//}))
	//b.Run("read_only_parallel", benchmark(&TestModeParams{}))
	//b.Run("read_only_taraxa", benchmark(&TestModeParams{
	//	SequentialTx: concurrentSchedule.SequentialTransactions,
	//}))
	//b.Run("read_only_sequential", benchmark(&TestModeParams{
	//	SequentialTx: allTransactions,
	//}))
	//b.Run("taraxa_mode_single_db", benchmark(&TestModeParams{
	//	SequentialTx: concurrentSchedule.SequentialTransactions,
	//	DoCommits:    true,
	//}))
	//b.Run("taraxa_mode", benchmark(&TestModeParams{
	//	SequentialTx:          concurrentSchedule.SequentialTransactions,
	//	DoCommitsInSeparateDB: true,
	//}))
	b.Run("ethereum_mode_commit_async", benchmark(&TestModeParams{
		SequentialTx: allTransactions,
		DoCommits:    true,
	}))
	//b.Run("ethereum_mode_separate_db", benchmark(&TestModeParams{
	//	SequentialTx:          allTransactions,
	//	DoCommitsInSeparateDB: true,
	//}))
	b.Run("ethereum_mode_commit_async_separate_db", benchmark(&TestModeParams{
		SequentialTx:          allTransactions,
		DoCommitsInSeparateDB: true,
	}))
}
