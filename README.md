# DistributedAuctionSystem

### Set-up
- Call the following in separate terminals:
    - `go run  rmserver.go 0 true`
    - `go run  rmserver.go 1 false`
    - `go run  rmserver.go 2 false`
    - `go run  feserver.go 4001`
    - `go run  feserver.go 4002`
    - `go run  client.go 4001`
    - `go run  client.go 4002`

- To bid:
    1. `bid`
    2. `<bid amount>`, e.g. 20
- To get result: `result`
