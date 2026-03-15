# Hybrid Asset Allocation

The **Hybrid Asset Allocation** strategy was developed by [Wouter Keller](https://papers.ssrn.com/sol3/cf_dev/AbsByAuth.cfm?per_id=1935527) and [JW Keuning](https://papers.ssrn.com/sol3/cf_dev/AbsByAuth.cfm?per_id=2530815). It is based on their paper: [Dual and Canary Momentum with Rising Yields/Inflation: Hybrid Asset Allocation (HAA)](https://papers.ssrn.com/sol3/papers.cfm?abstract_id=4346906). HAA combines dual momentum (absolute + relative) with a single canary asset (TIP) for crash protection. It is designed to be much simpler than BAA while achieving competitive risk-adjusted returns with lower cash fractions.

## Rules

The strategy uses a single canary asset and an offensive universe:

**HAA-Balanced (G8/T4):**
- **Offensive**: SPY, IWM, VEA, VWO, VNQ, DBC, IEF, TLT
- **Canary**: TIP
- **Defensive**: BIL, IEF (pick whichever has higher momentum)

**HAA-Simple (U1/T1):**
- **Offensive**: SPY
- **Canary**: TIP
- **Defensive**: BIL, IEF (pick whichever has higher momentum)

1. On the last trading day of the month, compute the 13612U momentum for all assets:
   - Momentum = (RET(1) + RET(3) + RET(6) + RET(12)) / 4
   - where RET(n) = p0/pn - 1 (n-month return)
   - This is the unweighted average of the 1, 3, 6, and 12-month total returns.
2. Check the canary asset (TIP):
   - If TIP momentum is non-positive, allocate 100% to the defensive asset with the highest momentum (BIL or IEF)
3. **Balanced allocation** (TIP momentum positive):
   - Rank the 8 offensive assets by momentum
   - Select the Top 4 (half the universe)
   - For each: if its momentum is positive, allocate 25% to it; if non-positive, allocate that 25% to the best defensive asset (BIL or IEF)
4. **Simple allocation** (TIP momentum positive):
   - If SPY momentum is also positive, allocate 100% to SPY
   - Otherwise, allocate 100% to the best defensive asset (BIL or IEF)
5. Hold all positions until the close of the following month.

## Assets Typically Held

| Ticker | Name                                                | Sector                              |
| ------ | --------------------------------------------------- | ----------------------------------- |
| SPY    | SPDR S&P 500 ETF                                    | Equity, U.S., Large Cap             |
| IWM    | iShares Russell 2000 ETF                            | Equity, U.S., Small Cap             |
| VEA    | Vanguard FTSE Developed Markets ETF                 | Equity, Developed Markets           |
| VWO    | Vanguard FTSE Emerging Markets ETF                  | Equity, Emerging Markets            |
| VNQ    | Vanguard Real Estate Index Fund ETF                 | Real Estate, U.S.                   |
| DBC    | Invesco DB Commodity Index Tracking Fund            | Commodity, Diversified              |
| IEF    | iShares 7-10 Year Treasury Bond ETF                 | Bond, U.S., Intermediate-Term       |
| TLT    | iShares 20+ Year Treasury Bond ETF                  | Bond, U.S., Long-Term               |
| TIP    | iShares TIPS Bond ETF                               | Bond, U.S., Inflation-Protected     |
| BIL    | SPDR Bloomberg 1-3 Month T-Bill ETF                 | Bond, U.S., Short-Term              |
