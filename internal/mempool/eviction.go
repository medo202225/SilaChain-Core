package mempool

func EvictAll(p *Pool) {
	if p == nil {
		return
	}
	p.Clear()
}
