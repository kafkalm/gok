# gok
a lightweight goroutinue pool
## Quickstart
```go
// init a Gok
gok := NewGok(100,10)

// Add a Gokit
errBus := gok.AddGokitFunc(func() error {
    fmt.Println(1)
    return nil
},"print","print-1")

// Start Gok
gok.Run()

// Read From ErrorBus
for {
    err := errBus.Next()
    t.Log(err)
}
```