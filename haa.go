// Copyright 2021-2026
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/penny-vault/pvbt/asset"
	"github.com/penny-vault/pvbt/data"
	"github.com/penny-vault/pvbt/engine"
	"github.com/penny-vault/pvbt/portfolio"
	"github.com/penny-vault/pvbt/tradecron"
	"github.com/penny-vault/pvbt/universe"
)

//go:embed README.md
var description string

type HybridAssetAllocation struct {
	OffensiveUniverse universe.Universe `pvbt:"offensive-universe" desc:"Offensive (risky) assets to select from" default:"SPY,IWM,VEA,VWO,VNQ,DBC,IEF,TLT" suggest:"HAA-Balanced=SPY,IWM,VEA,VWO,VNQ,DBC,IEF,TLT|HAA-Simple=SPY"`
	DefensiveUniverse universe.Universe `pvbt:"defensive-universe" desc:"Defensive assets for risk-off periods" default:"BIL,IEF" suggest:"HAA-Balanced=BIL,IEF|HAA-Simple=BIL,IEF"`
	CanaryUniverse    universe.Universe `pvbt:"canary-universe" desc:"Single canary asset for crash protection" default:"TIP" suggest:"HAA-Balanced=TIP|HAA-Simple=TIP"`
	TopX              int               `pvbt:"top-x" desc:"Number of top offensive assets to select (half the universe)" default:"4" suggest:"HAA-Balanced=4|HAA-Simple=1"`
}

func (s *HybridAssetAllocation) Name() string {
	return "Hybrid Asset Allocation"
}

func (s *HybridAssetAllocation) Setup(eng *engine.Engine) {
	tc, err := tradecron.New("@monthend", tradecron.MarketHours{Open: 930, Close: 1600})
	if err != nil {
		panic(err)
	}

	eng.Schedule(tc)
	eng.SetBenchmark(eng.Asset("VFINX"))
	eng.RiskFreeAsset(eng.Asset("DGS3MO"))
}

func (s *HybridAssetAllocation) Describe() engine.StrategyDescription {
	return engine.StrategyDescription{
		ShortCode:   "haa",
		Description: description,
		Source:      "https://papers.ssrn.com/sol3/papers.cfm?abstract_id=4346906",
		Version:     "1.0.0",
		VersionDate: time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC),
	}
}

func (s *HybridAssetAllocation) Compute(ctx context.Context, eng *engine.Engine, strategyPortfolio portfolio.Portfolio) error {
	// 1. Fetch 12-month window of monthly close prices for all universes.
	offensiveDF, err := s.OffensiveUniverse.Window(ctx, portfolio.Months(12), data.MetricClose)
	if err != nil {
		return fmt.Errorf("failed to fetch offensive universe prices: %w", err)
	}

	defensiveDF, err := s.DefensiveUniverse.Window(ctx, portfolio.Months(12), data.MetricClose)
	if err != nil {
		return fmt.Errorf("failed to fetch defensive universe prices: %w", err)
	}

	canaryDF, err := s.CanaryUniverse.Window(ctx, portfolio.Months(12), data.MetricClose)
	if err != nil {
		return fmt.Errorf("failed to fetch canary universe prices: %w", err)
	}

	// 2. Downsample to monthly frequency.
	offensiveMonthly := offensiveDF.Downsample(data.Monthly).Last()
	defensiveMonthly := defensiveDF.Downsample(data.Monthly).Last()
	canaryMonthly := canaryDF.Downsample(data.Monthly).Last()

	// Need at least 13 rows for Pct(12) to produce valid values.
	if offensiveMonthly.Len() < 13 || defensiveMonthly.Len() < 13 || canaryMonthly.Len() < 13 {
		return nil
	}

	// 3. Compute 13612U momentum (unweighted average of 1, 3, 6, 12-month returns) for all universes.
	offensiveMom := momentum13612U(offensiveMonthly)
	offensiveMom = offensiveMom.Drop(math.NaN()).Last()

	defensiveMom := momentum13612U(defensiveMonthly)
	defensiveMom = defensiveMom.Drop(math.NaN()).Last()

	canaryMom := momentum13612U(canaryMonthly)
	canaryMom = canaryMom.Drop(math.NaN()).Last()

	if offensiveMom.Len() == 0 || defensiveMom.Len() == 0 || canaryMom.Len() == 0 {
		return nil
	}

	offensiveMom.Annotate(strategyPortfolio)
	defensiveMom.Annotate(strategyPortfolio)
	canaryMom.Annotate(strategyPortfolio)

	// 4. Find best defensive (cash) asset by momentum.
	bestCash, bestCashScore := bestByMomentum(defensiveMom)

	ts := eng.CurrentDate().Unix()

	// 5. Check canary: if ANY canary asset has non-positive momentum, go 100% defensive.
	canaryBad := false

	for _, a := range canaryMom.AssetList() {
		if canaryMom.Value(a, data.MetricClose) <= 0 {
			canaryBad = true
			break
		}
	}

	regime := "offensive"
	if canaryBad {
		regime = "defensive"
	}

	strategyPortfolio.Annotate(ts, "regime", regime)
	strategyPortfolio.Annotate(ts, "best-cash", bestCash.Ticker)

	_ = bestCashScore

	members := make(map[asset.Asset]float64)

	var justification string

	if canaryBad {
		// 100% to best cash asset.
		members[bestCash] = 1.0
		justification = fmt.Sprintf("canary bad: 100%% %s", bestCash.Ticker)
	} else {
		// 6. Rank offensive assets by momentum, select TopX.
		type assetScore struct {
			a     asset.Asset
			score float64
		}

		var scores []assetScore

		for _, a := range offensiveMom.AssetList() {
			scores = append(scores, assetScore{a: a, score: offensiveMom.Value(a, data.MetricClose)})
		}

		sort.Slice(scores, func(i, j int) bool {
			return scores[i].score > scores[j].score
		})

		topX := s.TopX
		if topX > len(scores) {
			topX = len(scores)
		}

		weight := 1.0 / float64(topX)

		// 7. For each TopX asset: if momentum positive, allocate; else replace with best cash.
		for _, sc := range scores[:topX] {
			if sc.score > 0 {
				members[sc.a] += weight
			} else {
				members[bestCash] += weight
			}
		}

		justification = fmt.Sprintf("offensive: top %d, cash=%s", topX, bestCash.Ticker)
	}

	strategyPortfolio.Annotate(ts, "justification", justification)

	allocation := portfolio.Allocation{
		Date:          eng.CurrentDate(),
		Members:       members,
		Justification: justification,
	}

	if err := strategyPortfolio.RebalanceTo(ctx, allocation); err != nil {
		return fmt.Errorf("rebalance failed: %w", err)
	}

	return nil
}

// momentum13612U computes the unweighted 13612U momentum:
//
//	(RET(1) + RET(3) + RET(6) + RET(12)) / 4
//
// where RET(n) = p0/pn - 1 (n-month return).
// This is the unweighted average of the 1, 3, 6, and 12-month total returns.
func momentum13612U(df *data.DataFrame) *data.DataFrame {
	ret1 := df.Pct(1)
	ret3 := df.Pct(3)
	ret6 := df.Pct(6)
	ret12 := df.Pct(12)

	return ret1.Add(ret3).Add(ret6).Add(ret12).DivScalar(4)
}

// bestByMomentum returns the asset with the highest momentum score from a DataFrame.
func bestByMomentum(mom *data.DataFrame) (asset.Asset, float64) {
	var best asset.Asset

	bestScore := math.Inf(-1)

	for _, a := range mom.AssetList() {
		val := mom.Value(a, data.MetricClose)
		if val > bestScore {
			bestScore = val
			best = a
		}
	}

	return best, bestScore
}
