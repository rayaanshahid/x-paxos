package main
import (
    "fmt"
    "net"
    "os"
    "bufio"
    "sync"
    "encoding/gob"
    "time"
    "x-paxos/definitions"
    //"math"
    //"bytes"
    //"strconv"
)
var wg = sync.WaitGroup{}
/*type Replicate struct{
  Operation string
  TimeStamp int
  ClientId string
}
type Reply struct {
  SequenceNum int
  ViewNum int
  ClientId string
  replicate Replicate
}*/
var count int = 0

func connectwithall(ports []string) []net.Conn{
    Conns := make([]net.Conn, 5)
    //var Conns []net.Conn
    var OutgoingConnections []net.Conn
    var err error
    for i:=0;i<5;i++ {
        for{
            Conns[i], err = net.Dial("tcp", ports[i])
            if err==nil{
                break
            }
        }
        fmt.Println("connected with Outgoing: ", i,"conn : ",Conns[i])
        OutgoingConnections=append(OutgoingConnections,Conns[i])
    }
    return OutgoingConnections
}
/*func connectwithall(ports []string,portNum int) []net.Conn{
  var Connections []net.Conn
  service := ports[portNum]
  listener, err := net.Listen("tcp", service)
  checkError(err)
  for i:=0;i<5;i++{
    fmt.Println("listening for Node : ",i)
    conn, err := listener.Accept()
    if err == nil {
      Connections=append(Connections,conn)
    }
  }
  return Connections
}*/
/*func receiveReply(Conns []net.Conn) {
  for index:=0;index<5;index++ {
    wg.Add(1)
    go func(index int) {
      for {
        //receiving the value
        for i:=0;i<3;i++ {
            var value definitions.Reply
            err := gob.NewDecoder(Conns[index]).Decode(&value)
            if err != nil {
                //fmt.Println("Response error : ",err.Error())
                //break // connection already closed by client
            }else{
               fmt.Println("Response : ",value)
            }
        }
      }//infinite loop
      wg.Done()
    }(index)
  } // i = 1,2,3
}*/
func AutoMode(Conns []net.Conn,ClientID string){
  var timeList []int
  //var timeStamp int = 0
  for i:=1;i<=60;i++{

      var replicate definitions.Replicate
      if i >= 0 && i <= 20 {
        replicate=definitions.Replicate{
            Operation: "Operation 1",
            TimeStamp:  i,
            ClientId: ClientID,
        }
      }else if i >= 21 && i <= 40 {
        replicate=definitions.Replicate{
            Operation: "Operation 2",
            TimeStamp:  i,
            ClientId: ClientID,
        }
      }else if i >= 41 && i <= 60 {
        replicate=definitions.Replicate{
            Operation: "Operation 3",
            TimeStamp:  i,
            ClientId: ClientID,
        }
      }
      //start ticker
      Ticker:= time.NewTicker(time.Nanosecond)
      wg.Add(1)
      go TickerCounter(Ticker)
      //sending the value
      for i:=0;i<3;i++ {
          err1 := gob.NewEncoder(Conns[i]).Encode(replicate)
          if err1 != nil {
              fmt.Println("err while write:",err1.Error())
          }
      }
      for i:=0;i<3;i++ {
        var value definitions.Reply
        err := gob.NewDecoder(Conns[i]).Decode(&value)
        if err != nil {
            fmt.Println("Response error : ",err.Error())
        }else{
           fmt.Println("Response : ",value)
        }
      }
      //time.Sleep(1 * time.Second)
      Ticker.Stop()
      fmt.Println("round trip time in NS : ",count)
      timeList = append(timeList, count)
      count=0
      //time.Sleep(10*time.Nanosecond)
  }
  fmt.Println("All times : ",timeList)
  wg.Done()
}

func TickerCounter(Ticker *time.Ticker){
    for {
        select {
            case <-Ticker.C:
                count++
            default:
                //break
        }
    }
    wg.Done()
}

func main() {
    fmt.Println("Choose Client Id ! ")
    fmt.Print("Enter 1, 2 or 3 : ")
        var i int
    _, _ = fmt.Scanf("%d", &i)
    i--

    var ports =[]string {"localhost:1300","localhost:1301","localhost:1302","localhost:1303","localhost:1304"}
    //var ports =[]string {"pitter22:1300","pitter25:1301","pitter29:1302"}
    var Ids =[]string {"client1","client2","client3"}
    ClientId:=Ids[i]

    Conns:=connectwithall(ports)
    //go receiveReply(Conns)
    reader := bufio.NewReader(os.Stdin)
    fmt.Println("Start Operation (y/n) : ")
    Op, _ := reader.ReadString('\n')
    if Op == "y\n" {
      AutoMode(Conns,ClientId)
    }else {
      fmt.Println("Exiting !!!")
    }
    wg.Wait()
}

func checkError(err error) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
        //os.Exit(1)
    }
}
