package main
import (
    "fmt"
    "net"
    "os"
    "bufio"
    "sync"
    "encoding/gob"
    "time"
    "github.com/definitions"
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
func receiveReply(Conns []net.Conn) {
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
}
func ManualMode(Conns []net.Conn,ClientID string){
  var timeStamp int = 0
  for{
    reader := bufio.NewReader(os.Stdin)
    fmt.Println("Enter Operation Type ! ")
    fmt.Println("1. Operation 1 ")
    fmt.Println("2. Operation 2 ")
    fmt.Println("3. Operation 3 ")
    fmt.Print("Enter 1,2 or 3 : ")
    Op, _ := reader.ReadString('\n')
    if Op == "1\n" || Op == "2\n" || Op == "3\n" {
      var replicate definitions.Replicate
      if Op == "1\n" {
        timeStamp+=1
        replicate=definitions.Replicate{
            Operation: "Operation 1",
            TimeStamp:  timeStamp,
            ClientId: ClientID,
        }
      }else if Op == "2\n" {
        timeStamp+=1
        replicate=definitions.Replicate{
            Operation: "Operation 2",
            TimeStamp:  timeStamp,
            ClientId: ClientID,
        }
      }else if Op == "3\n" {
        timeStamp+=1
        replicate=definitions.Replicate{
            Operation: "Operation 3",
            TimeStamp:  timeStamp,
            ClientId: ClientID,
        }
      }
      //start ticker
      Ticker:= time.NewTicker(time.Nanosecond)
      wg.Add(1)
      go TickerCounter(Ticker)
      //sending the value
      for i:=0;i<5;i++ {
          err1 := gob.NewEncoder(Conns[i]).Encode(replicate)
          if err1 != nil {
              fmt.Println("err while write:",err1.Error())
          }
      }
      time.Sleep(1 * time.Second)
      Ticker.Stop()
      //fmt.Println("round trip time in NS : ",count)
      count=0
    }else {
      fmt.Print("!!! Enter Correct Operation !!!")
    }

  }
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
    var Ids =[]string {"client1","client2","client3"}
    ClientId:=Ids[i]

    Conns:=connectwithall(ports)
    go receiveReply(Conns)
    ManualMode(Conns,ClientId)

    wg.Wait()
}

func checkError(err error) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
        //os.Exit(1)
    }
}
