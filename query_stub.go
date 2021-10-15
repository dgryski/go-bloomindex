//go:build amd64 && !tinygo
// +build amd64,!tinygo

package bloomindex

//go:generate go run asm.go -out query_amd64.s
//go:noescape
func queryCore(r *bitrow, bits []bitrow, hashes []uint32)
