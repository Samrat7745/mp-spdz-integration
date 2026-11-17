# Carbyne Stack MP-SPDZ Integration Utilities (Go)

Go port of [mp-spdz-integration](https://github.com/carbynestack/mp-spdz-integration):
utilities for using [MP-SPDZ](https://github.com/data61/MP-SPDZ) in the Carbyne
Stack microservices. It converts big integers to and from the MP-SPDZ internal
(Montgomery, 16-byte) representation. It uses only the standard library.

## Usage

```go
import mpspdz "github.com/Samrat7745/mp-spdz-integration"

u, err := mpspdz.New(prime, r, rInv) // *big.Int parameters
gfp, err := u.ToGfp(big.NewInt(42))  // 16-byte MP-SPDZ representation
v, err := u.FromGfp(gfp)             // back to *big.Int
```

## Testing

```sh
go test ./...
```

## License

Apache License 2.0. See the [LICENSE](LICENSE) file for details.
