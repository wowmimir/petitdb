package dispatcher

type Command struct {
    Name string   // e.g., "SET"
    Args [][]byte // Raw arguments from RESP
}