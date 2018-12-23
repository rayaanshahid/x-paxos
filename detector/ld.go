package detector

import(
	//"fmt"
	"net"
)

func Ld(quorumList []int) int{
	//quorum,quorumList := findQuorum(clusterOfNodes)
	min := 100
	for i:=0;i<len(quorumList);i++ {
		if min > quorumList[i] {
			min = quorumList[i]
		}
	}
	return min
}
func FindQuorumForIntegers(clusterOfNodes []int, NumOfProcesses int)(int,[]int){
	var quorum int
	var quorumList []int
	quorum = NumOfProcesses/2 + 1
	for i:=0;i<quorum;i++ {
		quorumList=append(quorumList,clusterOfNodes[i])
	}
	return quorum,quorumList
}
func FindQuorumForConnections(Connections []net.Conn)([]net.Conn){
	var quorum int
	var quorumList []net.Conn
	quorum = len(Connections) / 2 + 1
	for i:=0;i<quorum-1;i++ { // one less connection than quorum
		quorumList=append(quorumList,Connections[i])
	}
	return quorumList
}
