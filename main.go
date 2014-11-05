package main
import(
    "fmt"
    "net"
    "strings"
)

var(
    name string
    chsender chan int
    chreciver chan int
)
func main(){
    fmt.Printf("名字？")
    fmt.Scanln(&name)

    localAddress, err := net.ResolveUDPAddr("udp", ":0")
    mcastAddress, err := net.ResolveUDPAddr("udp", "224.0.1.60:8765")

    mcastConn, err := net.ListenMulticastUDP("udp", nil, mcastAddress)
    localConn, err := net.ListenUDP("udp", localAddress)
    check(err)

    go sender(chsender, localConn, mcastAddress)
    go receiver(chreciver, mcastConn)
    <-chsender
    <-chreciver
}

func sender(ch chan int, conn *net.UDPConn, addr *net.UDPAddr){
    var words string
    for{
        fmt.Print("<- ")
        fmt.Scanf("%s", &words)
        fmt.Print("\r")
        message := fmt.Sprintf("%s: %s", name, words)
        buffer := make([]byte, 512)
        copy(buffer, []byte(message))
        _, err := conn.WriteToUDP(buffer, addr)
        check(err)
    }
    ch<-1
}

func receiver(ch chan int, conn *net.UDPConn) {
    for{
        b:= make([]byte, 256)
        _, _, err := conn.ReadFromUDP(b)
        check(err)
        msg := string(b)
        from := strings.Split(msg, ":")[0]
        if from == name{
            continue
        }
        fmt.Println("\r->", msg)
        fmt.Print("<- ")
    }
    ch<-1
}

func check(err error){
    if err != nil{
        fmt.Println(err)
        panic(err)
    }
}
