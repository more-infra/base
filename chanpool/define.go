package chanpool

const (
	groupMaxCount = 65536
)

type SelectResult string

func (r SelectResult) String() string {
	return string(r)
}

const (
	SelectQuitReturned    SelectResult = "quit"
	SelectRefreshReturned SelectResult = "refresh"
	SelectKeyReturned     SelectResult = "key"
)
