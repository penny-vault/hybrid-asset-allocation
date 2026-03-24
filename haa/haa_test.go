package haa_test

import (
	"context"
	"sort"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/penny-vault/hybrid-asset-allocation/haa"
	"github.com/penny-vault/pvbt/data"
	"github.com/penny-vault/pvbt/engine"
	"github.com/penny-vault/pvbt/portfolio"
)

var _ = Describe("HybridAssetAllocation", func() {
	var (
		ctx       context.Context
		snap      *data.SnapshotProvider
		nyc       *time.Location
		startDate time.Time
		endDate   time.Time
	)

	BeforeEach(func() {
		ctx = context.Background()

		var err error
		nyc, err = time.LoadLocation("America/New_York")
		Expect(err).NotTo(HaveOccurred())

		snap, err = data.NewSnapshotProvider("testdata/snapshot.db")
		Expect(err).NotTo(HaveOccurred())

		startDate = time.Date(2025, 1, 1, 0, 0, 0, 0, nyc)
		endDate = time.Date(2026, 3, 1, 0, 0, 0, 0, nyc)
	})

	AfterEach(func() {
		if snap != nil {
			snap.Close()
		}
	})

	runBacktest := func() portfolio.Portfolio {
		strategy := &haa.HybridAssetAllocation{}
		acct := portfolio.New(
			portfolio.WithCash(100000, startDate),
			portfolio.WithAllMetrics(),
		)

		eng := engine.New(strategy,
			engine.WithDataProvider(snap),
			engine.WithAssetProvider(snap),
			engine.WithAccount(acct),
		)

		result, err := eng.Backtest(ctx, startDate, endDate)
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	It("produces expected returns and risk metrics", func() {
		result := runBacktest()

		summary, err := result.Summary()
		Expect(err).NotTo(HaveOccurred())
		Expect(summary.TWRR).To(BeNumerically("~", 0.2246, 0.01))
		Expect(summary.MaxDrawdown).To(BeNumerically(">", -0.15), "max drawdown should be better than -15%")

		Expect(result.Value()).To(BeNumerically("~", 122455, 500))
	})

	It("trades all offensive universe assets", func() {
		result := runBacktest()
		txns := result.Transactions()

		tickers := map[string]bool{}
		for _, t := range txns {
			if t.Type == portfolio.BuyTransaction || t.Type == portfolio.SellTransaction {
				tickers[t.Asset.Ticker] = true
			}
		}

		Expect(tickers).To(HaveKey("SPY"))
		Expect(tickers).To(HaveKey("IWM"))
		Expect(tickers).To(HaveKey("VEA"))
		Expect(tickers).To(HaveKey("VWO"))
		Expect(tickers).To(HaveKey("VNQ"))
		Expect(tickers).To(HaveKey("DBC"))
		Expect(tickers).To(HaveKey("IEF"))
	})

	It("produces the expected trade sequence", func() {
		result := runBacktest()
		txns := result.Transactions()

		type trade struct {
			date   string
			txType portfolio.TransactionType
			ticker string
		}

		var trades []trade
		for _, t := range txns {
			if t.Type == portfolio.BuyTransaction || t.Type == portfolio.SellTransaction {
				trades = append(trades, trade{
					date:   t.Date.In(nyc).Format("2006-01-02"),
					txType: t.Type,
					ticker: t.Asset.Ticker,
				})
			}
		}

		// Sort trades by date, then type (sell before buy), then ticker for deterministic comparison.
		sort.Slice(trades, func(i, j int) bool {
			if trades[i].date != trades[j].date {
				return trades[i].date < trades[j].date
			}

			if trades[i].txType != trades[j].txType {
				return trades[i].txType > trades[j].txType // sell before buy
			}

			return trades[i].ticker < trades[j].ticker
		})

		expected := []trade{
			// 2025-01-31: offensive top 4 = SPY, IWM, VWO, VEA
			{"2025-01-31", portfolio.BuyTransaction, "IWM"},
			{"2025-01-31", portfolio.BuyTransaction, "SPY"},
			{"2025-01-31", portfolio.BuyTransaction, "VEA"},
			{"2025-01-31", portfolio.BuyTransaction, "VWO"},
			// 2025-02-28: top 4 = SPY, VWO, DBC, VNQ
			{"2025-02-28", portfolio.SellTransaction, "IWM"},
			{"2025-02-28", portfolio.SellTransaction, "VEA"},
			{"2025-02-28", portfolio.SellTransaction, "VWO"},
			{"2025-02-28", portfolio.BuyTransaction, "DBC"},
			{"2025-02-28", portfolio.BuyTransaction, "VNQ"},
			// 2025-03-31: top 4 = DBC, VWO, VEA, IEF
			{"2025-03-31", portfolio.SellTransaction, "DBC"},
			{"2025-03-31", portfolio.SellTransaction, "SPY"},
			{"2025-03-31", portfolio.SellTransaction, "VNQ"},
			{"2025-03-31", portfolio.SellTransaction, "VWO"},
			{"2025-03-31", portfolio.BuyTransaction, "IEF"},
			{"2025-03-31", portfolio.BuyTransaction, "VEA"},
			// 2025-04-30: top 4 = VEA, IEF, VWO, VNQ
			{"2025-04-30", portfolio.SellTransaction, "DBC"},
			{"2025-04-30", portfolio.SellTransaction, "IEF"},
			{"2025-04-30", portfolio.SellTransaction, "VEA"},
			{"2025-04-30", portfolio.SellTransaction, "VWO"},
			{"2025-04-30", portfolio.BuyTransaction, "VNQ"},
			// 2025-05-30: top 4 = VEA, VWO, SPY, IEF
			{"2025-05-30", portfolio.SellTransaction, "VEA"},
			{"2025-05-30", portfolio.SellTransaction, "VNQ"},
			{"2025-05-30", portfolio.SellTransaction, "VWO"},
			{"2025-05-30", portfolio.BuyTransaction, "IEF"},
			{"2025-05-30", portfolio.BuyTransaction, "SPY"},
			// 2025-06-30: top 4 = VEA, VWO, SPY, IWM
			{"2025-06-30", portfolio.SellTransaction, "IEF"},
			{"2025-06-30", portfolio.SellTransaction, "VWO"},
			{"2025-06-30", portfolio.BuyTransaction, "IWM"},
			{"2025-06-30", portfolio.BuyTransaction, "VEA"},
			// 2025-07-31: top 4 = VWO, SPY, VEA, DBC
			{"2025-07-31", portfolio.SellTransaction, "IWM"},
			{"2025-07-31", portfolio.BuyTransaction, "DBC"},
			{"2025-07-31", portfolio.BuyTransaction, "VEA"},
			{"2025-07-31", portfolio.BuyTransaction, "VWO"},
			// 2025-08-29: top 4 = VWO, VEA, IWM, SPY
			{"2025-08-29", portfolio.SellTransaction, "DBC"},
			{"2025-08-29", portfolio.SellTransaction, "VEA"},
			{"2025-08-29", portfolio.SellTransaction, "VWO"},
			{"2025-08-29", portfolio.BuyTransaction, "IWM"},
			{"2025-08-29", portfolio.BuyTransaction, "VEA"},
			{"2025-08-29", portfolio.BuyTransaction, "VWO"},
			// 2025-09-30: top 4 = VWO, SPY, IWM, VEA (minor rebalance)
			{"2025-09-30", portfolio.SellTransaction, "VWO"},
			{"2025-09-30", portfolio.BuyTransaction, "IWM"},
			{"2025-09-30", portfolio.BuyTransaction, "VEA"},
			{"2025-09-30", portfolio.BuyTransaction, "VWO"},
			// 2025-10-31: top 4 = VWO, IWM, SPY, VEA (minor rebalance)
			{"2025-10-31", portfolio.SellTransaction, "VEA"},
			{"2025-10-31", portfolio.BuyTransaction, "VEA"},
			{"2025-10-31", portfolio.BuyTransaction, "VWO"},
			// 2025-11-28: top 4 = VEA, VWO, SPY, IWM (minor rebalance)
			{"2025-11-28", portfolio.SellTransaction, "VEA"},
			{"2025-11-28", portfolio.BuyTransaction, "VEA"},
			{"2025-11-28", portfolio.BuyTransaction, "VWO"},
			// 2025-12-31: top 4 = VEA, VWO, SPY, IWM (minor rebalance)
			{"2025-12-31", portfolio.SellTransaction, "VEA"},
			{"2025-12-31", portfolio.BuyTransaction, "IWM"},
			{"2025-12-31", portfolio.BuyTransaction, "VWO"},
			// 2026-01-30: top 4 = VEA, VWO, DBC, IWM
			{"2026-01-30", portfolio.SellTransaction, "SPY"},
			{"2026-01-30", portfolio.SellTransaction, "VEA"},
			{"2026-01-30", portfolio.SellTransaction, "VWO"},
			{"2026-01-30", portfolio.BuyTransaction, "DBC"},
			// 2026-02-27: top 4 = VEA, VWO, DBC, IWM (minor rebalance)
			{"2026-02-27", portfolio.SellTransaction, "VEA"},
			{"2026-02-27", portfolio.BuyTransaction, "DBC"},
			{"2026-02-27", portfolio.BuyTransaction, "IWM"},
		}

		Expect(trades).To(HaveLen(len(expected)), "trade count mismatch")
		for i, exp := range expected {
			Expect(trades[i].date).To(Equal(exp.date), "trade %d date", i)
			Expect(trades[i].txType).To(Equal(exp.txType), "trade %d type", i)
			Expect(trades[i].ticker).To(Equal(exp.ticker), "trade %d ticker", i)
		}
	})
})
