// +build amd64

package bloomindex

//go:generate go run asm.go -out query_amd64.s
//go:noescape
func queryCore(r *bitrow, bits []bitrow, hashes []uint32)
