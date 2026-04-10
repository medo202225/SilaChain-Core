package types

func ReceiptCount(receipts []Receipt) uint64 {
	return uint64(len(receipts))
}
