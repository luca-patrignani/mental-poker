package common

type Player struct {
	MyRank int
}

func (p Player) BroadcastAllToAll(bufferSend []interface{}) (bufferRecv []interface{}) {
	bufferRecv = nil
	return
}

func (p Player) Broadcast(bufferSend interface{}, root int) (bufferRecv interface{}, err error) {
	if root == p.MyRank {
		bufferRecv = bufferSend
		// send
	} else {
		bufferRecv = nil
		// receive
	}
	return
}
