package definitions

import (
  //"crypto/rsa"
)

type Replicate struct{
  Operation string
  TimeStamp int
  ClientId string
}
type Prepare struct{
  RequestHash []byte
  SequenceNum int
  ViewNum int
}
type Commit struct{
  RequestHash []byte
  SequenceNum int
  ViewNum int
}
type PrepRequest struct {
  Rep Replicate
  Prep Prepare
}
type Reply struct {
  SequenceNum int
  ViewNum int
  ClientId string
  Rep Replicate
}
type Suspect struct {
  ViewNum int
  ClientNum int
}
type ViewChange struct {
  ViewNum int
  ClientNum int
  CommitLog []Commit
}
type ViewChangeFinal struct {
  ViewNum int
  ClientNum int
  ViewChangeLog []ViewChange
}
type Msg struct{
	MsgType int
	Msg []byte
  ClientNum int
  Signature []byte
}
type Channels struct{
  ReplicateOut chan Replicate
  PrepareOut chan Prepare
	CommitOut chan Commit
  MessageOut chan Msg
}
func NewChannels() Channels {
  return Channels{
    ReplicateOut : make(chan Replicate, 16),
    PrepareOut : make(chan Prepare, 16),
    CommitOut :  make(chan Commit, 16),
    MessageOut : make(chan Msg, 16),
  }
}
