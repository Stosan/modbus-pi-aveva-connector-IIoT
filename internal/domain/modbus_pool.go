package domain

import (
    "context"
    "fmt"
    "time"

    "github.com/goburrow/modbus"
)

// modbusPool manages a fixed set of Modbus connections to a single gateway.
// Modbus TCP supports multiple concurrent connections to the same slave —
// the pool lets goroutines grab a connection without serializing on one.
type modbusPool struct {
    conns chan modbus.Client
}

func newModbusPool(address string, slaveID byte, size int) (*modbusPool, error) {
    pool := &modbusPool{
        conns: make(chan modbus.Client, size),
    }
    for range size {
        h := modbus.NewTCPClientHandler(address)
        h.Timeout = 10 * time.Second
        h.SlaveId = slaveID
        if err := h.Connect(); err != nil {
            return nil, fmt.Errorf("pool init connect failed: %w", err)
        }
        pool.conns <- modbus.NewClient(h)
    }
    return pool, nil
}

// Acquire blocks until a connection is available or ctx is cancelled.
func (p *modbusPool) Acquire(ctx context.Context) (modbus.Client, error) {
    select {
    case conn := <-p.conns:
        return conn, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

// Release returns a connection to the pool.
func (p *modbusPool) Release(conn modbus.Client) {
    p.conns <- conn
}