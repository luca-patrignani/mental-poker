package common

type Player struct {
	MyRank int
}

func (p Player) BroadcastAllToAll(bufferSend string) (bufferRecv []string) {
	bufferRecv = nil
	return
}

func (p Player) Broadcast(bufferSend string, root int) (bufferRecv string, err error) {
	if root == p.MyRank {
		bufferRecv = bufferSend
		// send
	} else {
		bufferRecv = ""
		// receive
	}
	return
}
