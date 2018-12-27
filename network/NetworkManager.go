package network

import (
    "fmt"
    "net"
    //"net/http"
    "os"
    "time"
    "bytes"
    "x-paxos/detector"
    "x-paxos/definitions"
    "sync"
    "encoding/gob"
    //"strconv"
    "crypto/rsa"
)

type NetworkManager struct{
  IncommingConnections []net.Conn
  OutgoingConnections []net.Conn
  Connections []net.Conn
  QuorumConnections []net.Conn
  PublicKey map[int]rsa.PublicKey
  ClientConnections []net.Conn
  ClientConnectionMap map[int]net.Conn
  clientPorts []string
  ports []string
  portNum int
  NumOfProcesses int
  ClusterOfNodes []int
  Leader int
  quorumPIDs []int
  quorum int
  Wg sync.WaitGroup
  MessageOut chan definitions.Msg
  ClientId string
  replicate definitions.Replicate
  commit definitions.Commit
  commitLog []definitions.Commit
  View map[int][]int
  ViewNum int
  QuorumConnectionsMap map[int]net.Conn
  ViewChangeLog []definitions.ViewChange
  ViewChangeFinalLog []definitions.ViewChangeFinal
}

func NewNetworkManager(ports []string,portNum int,NumOfProcesses int,clientPorts []string,clusterOfNodes []int,MessageOut chan definitions.Msg) *NetworkManager{
  return &NetworkManager{
    ClientConnectionMap : make(map[int]net.Conn),
    PublicKey: make(map[int]rsa.PublicKey),
    clientPorts:  clientPorts,
		ports: ports,
    portNum: portNum,
    NumOfProcesses: NumOfProcesses,
    ClusterOfNodes: clusterOfNodes,
    Leader: -1,
    quorum: -1,
    MessageOut: MessageOut,
    View: make(map[int][]int),
    ViewNum: 1,
    QuorumConnectionsMap: make(map[int]net.Conn),
	}
}

func (nm *NetworkManager)ListenForConnections(){
  //fmt.Println("listening function called")
  nm.Wg.Add(1)
  // Incomming Connections Handled In this Go Routine
  go func() {
    service := nm.ports[nm.portNum]
    listener, err := net.Listen("tcp", service)
    checkError(err)
    for i:=0;i<nm.portNum;i++{
      fmt.Println("listening ...")
      conn, err := listener.Accept()
      if err == nil {
        nm.IncommingConnections=append(nm.IncommingConnections,conn)
        fmt.Println("connected with Incomming: ",conn)
      }
    }
    nm.Wg.Done()
  }()
}

func (nm *NetworkManager)MakeConnections(){
  Conns := make([]net.Conn, 5)
  var err error
  var j int = 0
	for i:=nm.portNum+1;i<nm.NumOfProcesses;i++ {
    for{
      Conns[j], err = net.Dial("tcp", nm.ports[i])
      if err==nil{
          break
      }
    }
    fmt.Println("connected with Outgoing: ", j,"conn : ",Conns[j])
    nm.OutgoingConnections=append(nm.OutgoingConnections,Conns[j])
    j+=1
  }
}

func (nm *NetworkManager)MergeConnections() []net.Conn{
  for i:=0;i<len(nm.IncommingConnections);i++ {
    nm.Connections=append(nm.Connections,nm.IncommingConnections[i])
  }
  for i:=0;i<len(nm.OutgoingConnections);i++ {
    nm.Connections=append(nm.Connections,nm.OutgoingConnections[i])
  }
  /*j:=0
  for i:=0;i<len(nm.Connections);i++ {
    if i == nm.portNum {
      j+=1
    }
    nm.QuorumConnectionsMap[j] = nm.Connections[i]
    j+=1
  }
  fmt.Println("nm.QuorumConnectionsMap : ",nm.QuorumConnectionsMap)*/
  return nm.Connections
}

func (nm *NetworkManager)FindQuorumConnections() ([]net.Conn,int){
  nm.QuorumConnections = detector.FindQuorumForConnections(nm.Connections)
  nm.quorum,nm.quorumPIDs = detector.FindQuorumForIntegers(nm.ClusterOfNodes,nm.NumOfProcesses)
  //Elect the Leader
  nm.Leader = detector.Ld(nm.quorumPIDs)
  return nm.QuorumConnections,nm.Leader
}

func (nm *NetworkManager)ClientConnection(){
  service := nm.clientPorts[nm.portNum]
  listener, err := net.Listen("tcp", service)
  checkError(err)
  for i:=0;i<3;i++{
    fmt.Println("listening for client : ",i)
    conn, err := listener.Accept()
    if err == nil {
      nm.ClientConnections=append(nm.ClientConnections,conn)
      nm.ClientConnectionMap[i+1]=conn
      //fmt.Println("connected with Client : ",conn, "client connection map key:",i+1,"value",nm.ClientConnectionMap[i+1])
      //nm.Wg.Add(1)
      go nm.receiveReplicate(conn)
    }
  }
}
/*func (nm *NetworkManager)ClientConnection(){
  for i:=0;i<3;i++ {
      for{
          conn, err := net.Dial("tcp", nm.clientPorts[i])
          if err==nil{
            nm.ClientConnections=append(nm.ClientConnections,conn)
            nm.ClientConnectionMap[i+1]=conn
            go nm.receiveReplicate(conn)
            break
          }
      }
      fmt.Println("connected with Outgoing: ", i)
  }
}*/
func (nm *NetworkManager)receiveReplicate(conn net.Conn){
  //fmt.Println("listening for client replicate : ")
  for {
    var replicate definitions.Replicate
    err := gob.NewDecoder(conn).Decode(&replicate)
    if err == nil && nm.Leader == nm.portNum{
      fmt.Println("Replicate Message recieved : ",replicate)
      //nm.ClientId = replicate.ClientId
      message:=definitions.Msg{}
      var network bytes.Buffer
      err = gob.NewEncoder(&network).Encode(replicate)
      message.MsgType=1
      message.Msg= network.Bytes()
      nm.MessageOut <- message
    }
  }
  //nm.Wg.Done()
}

func (nm *NetworkManager)SendMessage(M definitions.Msg){
  for i:=0;i<len(nm.QuorumConnections);i++ {
    err1 := gob.NewEncoder(nm.QuorumConnections[i]).Encode(&M)
    if err1 != nil {
      fmt.Println(err1.Error())
    }else{
      //fmt.Println("Sent Message")
    }
  }
}
func (nm *NetworkManager)SendMessageToAll(M definitions.Msg){
  for i:=0;i<len(nm.Connections);i++ {
    err1 := gob.NewEncoder(nm.Connections[i]).Encode(&M)
    if err1 != nil {
      fmt.Println(err1.Error())
    }else{
      //fmt.Println("Sent Message")
    }
  }
}

func (nm *NetworkManager)ReceiveMessagesSendReply() {
  for index:=0;index<len(nm.Connections);index++ {
    nm.Wg.Add(1)
    go func(index int) {
      for {
        newMsg:=definitions.Msg{}
        err := gob.NewDecoder(nm.Connections[index]).Decode(&newMsg)
        if err != nil  {
          //fmt.Println("Error while receiving message : ",err.Error())
        }
        if newMsg.MsgType == 2 || newMsg.MsgType == 3 || newMsg.MsgType == 11 || newMsg.MsgType == 12 || newMsg.MsgType == 13{
          if newMsg.MsgType == 11 {
            fmt.Println("received suspect in netwrok layer!")
          }
          nm.MessageOut <- newMsg
        }else if newMsg.MsgType == 10 {
          nm.QuorumConnectionsMap[newMsg.ClientNum] = nm.Connections[index]
          //fmt.Println("Set the connections map : ",nm.QuorumConnectionsMap)
          nm.MessageOut <- newMsg
        }
      }//infinite loop
      nm.Wg.Done()
    }(index)
  } // i = 1,2,3
}

func (nm *NetworkManager)SetReplicate(replicate definitions.Replicate){
  nm.replicate = replicate
}
func (nm *NetworkManager)SetCommit(commit definitions.Commit){
  nm.commit = commit
  nm.commitLog = append(nm.commitLog,commit)
}
func (nm *NetworkManager)AddViewChangeLog(viewChange definitions.ViewChange){
  nm.ViewChangeLog = append(nm.ViewChangeLog,viewChange)
  fmt.Println("Received View change")
}
func (nm *NetworkManager)AddViewChangeFinalLog(viewChangeFinal definitions.ViewChangeFinal){
  nm.ViewChangeFinalLog = append(nm.ViewChangeFinalLog,viewChangeFinal)
  fmt.Println("Received View change final")
}
func (nm *NetworkManager)SetPublicKey(pubKeyMsg rsa.PublicKey,clientNum int){
  nm.PublicKey[clientNum] = pubKeyMsg
  //fmt.Println("Recieved public key",nm.PublicKey)
  fmt.Println("Set the connections map : ",nm.QuorumConnectionsMap)
}
func (nm *NetworkManager)GetPublicKey(clientNum int) rsa.PublicKey{
  return nm.PublicKey[clientNum]
}
func (nm *NetworkManager)SendReplyToClient(){
  reply:=definitions.Reply{SequenceNum: nm.commit.SequenceNum,ViewNum: nm.commit.ViewNum,ClientId: nm.replicate.ClientId,Rep: nm.replicate}
  var connection net.Conn
  if reply.ClientId == "client1" {
    fmt.Println("Sending reply to client 1",nm.ClientConnectionMap[1])
    connection = nm.ClientConnectionMap[1]
  }else if reply.ClientId == "client2" {
    fmt.Println("Sending reply to client 2",nm.ClientConnectionMap[2])
    connection = nm.ClientConnectionMap[2]
  }else if reply.ClientId == "client3" {
    fmt.Println("Sending reply to client 3",nm.ClientConnectionMap[3])
    connection = nm.ClientConnectionMap[3]
  }
  fmt.Println("")
  err := gob.NewEncoder(connection).Encode(reply)
  if err != nil {
    fmt.Println(err.Error())
  }
}
func (nm *NetworkManager)SuspectView(ViewNum int){ // using the view number sent for now
  nm.ViewNum = ViewNum + 1
  fmt.Println("new view number : ",nm.ViewNum)
  newView:=nm.View[nm.ViewNum] // {0,1,2},{0,1,3}, ...
  fmt.Println("new view : ",newView)
  nm.QuorumConnections = nm.QuorumConnections[:0] // deleting old quorum
  fmt.Println("deleted connection : ",nm.QuorumConnections)
  for i:=0;i<len(newView);i++ {
    candidates:=newView[i] // 0,1,2 or 0,1,3 ...
    fmt.Println("candidates : ",candidates)
    conn:=nm.QuorumConnectionsMap[candidates]
    if conn != nil {
      nm.QuorumConnections=append(nm.QuorumConnections,conn) //appending new connection in quorum
    }
  }
  nm.quorumPIDs = newView
  nm.Leader = detector.Ld(nm.quorumPIDs) // change leader
  fmt.Println("Leader",nm.Leader,"new Quorum Connection",nm.QuorumConnections)
  nm.ViewChange()
}
func (nm *NetworkManager)SetViewMap(View map[int][]int){
  nm.View = View
  fmt.Println("view map : ",nm.View)
}
func (nm *NetworkManager)ViewChange(){
  viewChange:= definitions.ViewChange{ViewNum: nm.ViewNum, ClientNum: nm.portNum, CommitLog: nm.commitLog}
  message:=definitions.Msg{}
  var network bytes.Buffer
  _ = gob.NewEncoder(&network).Encode(viewChange)
  message.MsgType=12
  message.Msg= network.Bytes()
  message.ClientNum=nm.portNum
  nm.SendMessageToAll(message)
  ProgressTicker := time.NewTicker(time.Second)
  count := 0
  go nm.timeOut(ProgressTicker,count)
}
func (nm *NetworkManager)timeOut(ProgressTicker *time.Ticker, count int){
  for {
    select {
    case <-ProgressTicker.C:
      count += 1
      fmt.Println("ticker on !!!")
      if /*len(nm.ViewChangeLog) == nm.NumOfProcesses ||*/ count == 2{
        ProgressTicker.Stop()
        nm.ViewChangeFinal()
      }
    }
  }
}
func (nm *NetworkManager)ViewChangeFinal(){
  viewChangeFinal:= definitions.ViewChangeFinal{ViewNum: nm.ViewNum, ClientNum: nm.portNum, ViewChangeLog: nm.ViewChangeLog}
  message:=definitions.Msg{}
  var network bytes.Buffer
  _ = gob.NewEncoder(&network).Encode(viewChangeFinal)
  message.MsgType=13
  message.Msg= network.Bytes()
  message.ClientNum=nm.portNum
  nm.SendMessage(message)
}
func checkError(err error) {
    if err != nil {
        fmt.Println("Checking Error")
        fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
    }
}
