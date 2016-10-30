// +build amd64

package bloomindex

//go:generate python -m peachpy.x86_64 query.py -S -o query_amd64.s -mabi=goasm
func queryCore(r *bitrow, bits []bitrow, hashes []uint16)
