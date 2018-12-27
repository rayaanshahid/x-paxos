package main

import (
    "fmt"
    //"net"
    //"net/http"
    "os"
    "time"
    "bytes"
    "crypto/sha256"
    "x-paxos/network"
    "x-paxos/definitions"
    //"github.com/detector"
    "sync"
    "encoding/gob"
    //"strconv"
    "crypto/rsa"
    "crypto/rand"
    "crypto"
    "io"
)
var wg = sync.WaitGroup{}
var countForReply int = 0
var sequenceNumber int = 0
var viewNumber int = 1
var CountForReply int = 0
var tickerCounter int = 0
var viewSuspected bool = false
var ProcessRunNumber int = 0
var View = make(map[int][]int)
func calculateHash(Msg []byte) []byte{
  hash := sha256.New()
  hash.Write(Msg)
  //fmt.Printf("%x", hash.Sum(nil))
  return hash.Sum(nil)
}

func signMsg(Msg []byte,rsaPrivateKey *rsa.PrivateKey,rng io.Reader) []byte {
  hashed := sha256.Sum256(Msg)
  signature, err := rsa.SignPKCS1v15(rng, rsaPrivateKey, crypto.SHA256, hashed[:])
  if err != nil {
          fmt.Fprintf(os.Stderr, "Error from signing: %s\n", err)
          //return
  }
  return signature
}
func verifyMsg(networkManager *network.NetworkManager,msg definitions.Msg,rng io.Reader) bool {
  publicKey:=networkManager.GetPublicKey(msg.ClientNum)
  Hashed := sha256.Sum256(msg.Msg)

  err1 := rsa.VerifyPKCS1v15(&publicKey, crypto.SHA256, Hashed[:], msg.Signature)
  if err1 != nil {
          fmt.Fprintf(os.Stderr, "Error from verification: %s\n", err1)
          return false
  }else {
  	fmt.Println("Signature Verified !!")
    return true
  }
}
func timeOut(networkManager *network.NetworkManager,ProgressTicker *time.Ticker,portNum int,rsaPrivateKey *rsa.PrivateKey,rng io.Reader){
  for {
    select {
    case <-ProgressTicker.C:
      fmt.Println("ticker on !!!")
      ProgressTicker.Stop()
      fmt.Println("counter for reply : ",CountForReply," viewSuspected : ", viewSuspected)
      if CountForReply != 0 {
        fmt.Println("counter for reply is not 0")
        if viewSuspected == false {
          fmt.Println("sending suspect")
          suspectView(networkManager,portNum,rsaPrivateKey,rng)
          networkManager.SuspectView(viewNumber)
          viewNumber += 1
          viewSuspected = true
          fmt.Println("viewSuspected = true")
        }
      }
    }
  }
}
func suspectView(networkManager *network.NetworkManager,portNum int,rsaPrivateKey *rsa.PrivateKey,rng io.Reader){
  fmt.Println("started view change")
  M:=definitions.Msg{}
  suspect:=definitions.Suspect{ViewNum: viewNumber, ClientNum: portNum}
  var network1 bytes.Buffer
  _ = gob.NewEncoder(&network1).Encode(suspect)
  M.MsgType=11
  M.Msg= network1.Bytes()
  M.ClientNum=portNum
  M.Signature=signMsg(M.Msg,rsaPrivateKey,rng)
  networkManager.SendMessageToAll(M)
}
func EncodeMessage(networkManager *network.NetworkManager,message definitions.Msg,rsaPrivateKey *rsa.PrivateKey,portNum int,rng io.Reader,quorumLength int,ProgressTicker *time.Ticker){
  M:=definitions.Msg{}
  if message.MsgType == 1 {
    ProgressTicker = time.NewTicker(2*time.Nanosecond)
    go timeOut(networkManager,ProgressTicker,portNum,rsaPrivateKey,rng)
    var replicate definitions.Replicate
    var network bytes.Buffer
    var network1 bytes.Buffer
    buff:=bytes.NewBuffer(message.Msg)
    err := gob.NewDecoder(buff).Decode(&replicate)
    fmt.Println("message received replicate :",replicate," , sending Prepare")
    networkManager.SetReplicate(replicate)
    err = gob.NewEncoder(&network).Encode(replicate)
    hash:=calculateHash(network.Bytes())
    sequenceNumber+=1
    prepare:=definitions.Prepare{RequestHash: hash,SequenceNum: sequenceNumber,ViewNum: viewNumber}
    prepRequest:=definitions.PrepRequest{Rep: replicate,Prep: prepare}
    err = gob.NewEncoder(&network1).Encode(prepRequest)
    if err != nil {
      fmt.Println(err.Error())
    }
    M.MsgType=2
    M.Msg= network1.Bytes()
    M.ClientNum=portNum
    M.Signature=signMsg(M.Msg,rsaPrivateKey,rng)
    networkManager.SendMessage(M)
  }else if message.MsgType == 2 {
    ProgressTicker = time.NewTicker(2*time.Nanosecond)
    go timeOut(networkManager,ProgressTicker,portNum,rsaPrivateKey,rng)
    boolean:=verifyMsg(networkManager,message,rng)
    if boolean {
      var prepRequest definitions.PrepRequest
      buff:=bytes.NewBuffer(message.Msg)
      err := gob.NewDecoder(buff).Decode(&prepRequest)
      fmt.Println("message received Prepare:",prepRequest," , sending commit")
      if err == nil{
        rep:=prepRequest.Rep
        prep:=prepRequest.Prep
        networkManager.SetReplicate(rep)
        //calculating hash to match
        var network bytes.Buffer
        err = gob.NewEncoder(&network).Encode(rep)
        hash:=calculateHash(network.Bytes())
        if ProcessRunNumber == 0 {
          ProcessRunNumber += 1
          sequenceNumber = prep.SequenceNum - 1
          fmt.Println("set new sequence number :",sequenceNumber)
        }
        if bytes.Compare(hash,prep.RequestHash) == 0 || prep.SequenceNum == sequenceNumber + 1 {
          fmt.Println("matched hash value")
          sequenceNumber+=1
          commit:=definitions.Commit{RequestHash: prep.RequestHash, SequenceNum: sequenceNumber, ViewNum: prep.ViewNum}
          var network1 bytes.Buffer
          err = gob.NewEncoder(&network1).Encode(commit)
          M.MsgType=3
          M.Msg= network1.Bytes()
          M.ClientNum=portNum
          M.Signature=signMsg(M.Msg,rsaPrivateKey,rng)
          networkManager.SendMessage(M)
          CountForReply += 1
        }
      }
    }
  }else if message.MsgType == 3 {
    boolean:=verifyMsg(networkManager,message,rng)
    if boolean {
      var commit definitions.Commit
      buff:=bytes.NewBuffer(message.Msg)
      err := gob.NewDecoder(buff).Decode(&commit)
      if err != nil {fmt.Println(err.Error())}
      fmt.Println("message received Commit : ",commit)
      networkManager.SetCommit(commit)
      CountForReply += 1
    }
  }else if message.MsgType == 10 {
    var pubKeyMsg rsa.PublicKey
    buff:=bytes.NewBuffer(message.Msg)
    err := gob.NewDecoder(buff).Decode(&pubKeyMsg)
    if err != nil {fmt.Println(err.Error())}
    //fmt.Println("message received Public Key : ",pubKeyMsg)
    networkManager.SetPublicKey(pubKeyMsg,message.ClientNum)
  }else if message.MsgType == 11 {
    var suspect definitions.Suspect
    buff:=bytes.NewBuffer(message.Msg)
    err := gob.NewDecoder(buff).Decode(&suspect)
    if err != nil {fmt.Println(err.Error())}
    fmt.Println("received suspect , viewSuspected : ",viewSuspected)
    if viewSuspected == false {
      fmt.Println("suspecting view")
      networkManager.SuspectView(suspect.ViewNum)
      viewNumber += 1
      viewSuspected = true
      fmt.Println("viewSuspected = true")
    }
  }else if message.MsgType == 12 {
    var viewChange definitions.ViewChange
    buff:=bytes.NewBuffer(message.Msg)
    err := gob.NewDecoder(buff).Decode(&viewChange)
    if err != nil {fmt.Println(err.Error())}
    networkManager.AddViewChangeLog(viewChange)
  }else if message.MsgType == 13 {
    var viewChangeFinal definitions.ViewChangeFinal
    buff:=bytes.NewBuffer(message.Msg)
    err := gob.NewDecoder(buff).Decode(&viewChangeFinal)
    if err != nil {fmt.Println(err.Error())}
    networkManager.AddViewChangeFinalLog(viewChangeFinal)
  }

  if CountForReply == quorumLength {
    CountForReply = 0
    viewSuspected = false // Still not sure if it should be here
    fmt.Println("viewSuspected = false")
    networkManager.SendReplyToClient()
  }
}
func receiveMessage(MessageOut chan definitions.Msg,networkManager *network.NetworkManager,rsaPrivateKey *rsa.PrivateKey,portNum int,rng io.Reader, quorumLength int,ProgressTicker *time.Ticker){
  for {
    select {
    case message := <- MessageOut:
      //fmt.Println("received replicate in main")
      EncodeMessage(networkManager,message,rsaPrivateKey,portNum,rng,quorumLength,ProgressTicker)
    }
  }
  wg.Done()
}
func sendPublicKeyToAll(networkManager *network.NetworkManager,rsaPrivateKey *rsa.PrivateKey,portNum int){
  M:=definitions.Msg{}
  var network bytes.Buffer
  err := gob.NewEncoder(&network).Encode(rsaPrivateKey.PublicKey)
  checkError(err)
  M.MsgType=10
  M.ClientNum = portNum
  M.Msg= network.Bytes()
  networkManager.SendMessageToAll(M)
}
func Handler(networkManager *network.NetworkManager,ports []string,portNum int,NumOfProcesses int,clusterOfNodes []int,clientPorts []string,MessageOut chan definitions.Msg,rsaPrivateKey *rsa.PrivateKey,rng io.Reader,ProgressTicker *time.Ticker){
  networkManager.ListenForConnections()
  networkManager.MakeConnections()
  networkManager.Wg.Wait()
  connections:=networkManager.MergeConnections()
  fmt.Println("Final Connections",connections)

  QuorumConnections,Leader:=networkManager.FindQuorumConnections()
  fmt.Println("leader is : ",Leader," And Quorum connections : ",QuorumConnections)
  networkManager.SetViewMap(View)
  wg.Add(1)
  go receiveMessage(MessageOut,networkManager,rsaPrivateKey,portNum,rng,len(QuorumConnections),ProgressTicker)
  networkManager.ReceiveMessagesSendReply()
  sendPublicKeyToAll(networkManager,rsaPrivateKey,portNum)
  networkManager.ClientConnection()
}

func checkError(err error) {
    if err != nil {
        fmt.Println("Checking Error")
        fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
    }
}
func setView(){
  View[1]= []int{0,1,2}
  View[2]= []int{0,1,3}
  View[3]= []int{0,1,4}
  View[4]= []int{0,2,3}
  View[5]= []int{0,2,4}
  View[6]= []int{0,3,4}
  View[7]= []int{1,2,3}
  View[8]= []int{1,2,4}
  View[9]= []int{1,3,4}
  View[10]= []int{2,3,4}
}
func main(){
  var ports =[]string {"localhost:1200","localhost:1201","localhost:1202","localhost:1203","localhost:1204","localhost:1205","localhost:1206","localhost:1207","localhost:1208"}
  //var ports =[]string {"pitter11:1200","pitter12:1201","pitter15:1202","pitter17:1203","pitter19:1204"}
  var clientPorts =[]string {"localhost:1300","localhost:1301","localhost:1302","localhost:1303","localhost:1304","localhost:1305","localhost:1306","localhost:1307","localhost:1308"}
  //var clientPorts =[]string {"pitter22:1300","pitter25:1301","pitter29:1302"}
  var clusterOfNodes = []int {0,1,2,3,4,5,6,7,8}
  var NumOfProcesses int
  var portNum int
  fmt.Print("Enter Number of Nodes: ")
  _, err:= fmt.Scan(&NumOfProcesses)
  for {
      fmt.Print("Enter port num 1,2,3 ... : ")
      _, err:= fmt.Scan(&portNum)
      if err != nil || portNum < 1 {
      }else {
          break
      }
  }
  setView()
  rng := rand.Reader
  rsaPrivateKey,err:=rsa.GenerateKey(rng,1024)
  if err != nil {
  	fmt.Println(err.Error())
  }
  var ProgressTicker *time.Ticker
  MessageOut:=make(chan definitions.Msg, 16)
  networkManager:=network.NewNetworkManager(ports,portNum-1,NumOfProcesses,clientPorts,clusterOfNodes,MessageOut)
  Handler(networkManager,ports,portNum-1,NumOfProcesses,clusterOfNodes,clientPorts,MessageOut,rsaPrivateKey,rng,ProgressTicker)
  fmt.Println("Waiting for end")
  wg.Wait()
}
