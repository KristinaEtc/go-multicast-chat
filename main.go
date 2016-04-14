package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	name      string
	chsender  chan int
	chreciver chan int
	userNames []string
)

type connStr struct {
	localAddress *net.UDPAddr
	mcastAddress *net.UDPAddr
	mcastConn    *net.UDPConn
	localConn    *net.UDPConn
}

const (
	commGetNicks   string = "GET_USERNAME"
	commMsg        string = "MSG"
	commMyNick     string = "MY_NICK"
	commNickExist  string = "NICK_EXIST"
	commChangeNick string = "CHANGE_NICK"
	//commLeave     string = "LEAVED"
	// commChangeNick string = "/NICK"
	//private string = "PRIVATE"
)

//func getMyIP() (addr [4]byte, err error) {
func getMyIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		//os.Stderr.WriteString("Oops: " + err.Error() + "\n")
		//os.Exit(1)
		return "", err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				//return os.Stdout.WriteString(ipnet.IP.String() + "\n")

				//copy(addr[:], ipnet.IP.To4())
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", err
}

var connection = connStr{}
var myRand *rand.Rand

func main() {

	s1 := rand.NewSource(time.Now().UnixNano())
	myRand = rand.New(s1)

	fmt.Printf("nickname？")
	fmt.Scanln(&name)

	myIP, err := getMyIP()

	connection = connStr{}

	connection.localAddress, err = net.ResolveUDPAddr("udp", myIP+":0")
	check(err)
	connection.mcastAddress, err = net.ResolveUDPAddr("udp", "224.0.1.60:8765")
	check(err)

	connection.mcastConn, err = net.ListenMulticastUDP("udp", nil, connection.mcastAddress)
	check(err)
	connection.localConn, err = net.ListenUDP("udp", connection.localAddress)
	check(err)

	//fmt.Println(connection.localConn.LocalAddr())

	//check(err)
	message := fmt.Sprintf("%s:%s", commMyNick, name)
	//fmt.Println("ip ", message)
	buffer := make([]byte, len(message))
	copy(buffer, []byte(message))
	_, err = connection.localConn.WriteToUDP(buffer, connection.mcastAddress)
	check(err)

	go sender(chsender, connection.localConn, connection.mcastAddress)
	go receiver(chreciver, connection.mcastConn)
	<-chsender
	<-chreciver
}

func sender(ch chan int, conn *net.UDPConn, addr *net.UDPAddr) {
	//var words string
	/*for {
		//fmt.Print("<- ")
		fmt.Scanf("%s", &words)
		fmt.Print("\r")
		message := fmt.Sprintf("%s:%s: %s", commMsg, name, words)
		buffer := make([]byte, len(message))
		copy(buffer, []byte(message))
		_, err := conn.WriteToUDP(buffer, addr)
		check(err)
		words = ""
		fmt.Print("<- ")
	}*/

	//распределенный консенсунс
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := fmt.Sprintf("%s:%s: %s", commMsg, name, scanner.Text())
		fmt.Print("\r")

		var msgParted []string
		msgParted = strings.Split(msg, " ")
		if _, command := msgParted[0]; command {
			if command == commChangeNick {
				msg = commChangeNick + msgParted[1]
			}
		}
		buffer := make([]byte, len(msg))
		copy(buffer, []byte(msg))
		_, err := conn.WriteToUDP(buffer, addr)
		check(err)
		fmt.Print("<- ")
	}
	fmt.Print("quit")
	ch <- 1
}

func receiver(ch chan int, conn *net.UDPConn) {
	for {
		b := make([]byte, 256)
		n, addr, err := conn.ReadFromUDP(b)
		check(err)
		//fmt.Printf("n:%d", n)
		rawMsg := string(b[:n])

		//fmt.Println("addr ", addr)

		var msg []string
		msg = strings.SplitN(rawMsg, ":", 3)
		if len(msg) < 1 {
			continue
		}

		//if msg[0] != commMsg && msg[0] != commMyNick {
		//	fmt.Println("rawMsg ", rawMsg)
		//}

		switch msg[0] { //check command type
		case commMsg:
			{
				if msg[1] == name {
					break
				}
				fmt.Printf("\r-> %s: %s\n", msg[1], msg[2])
				fmt.Print("<- ")
				//break
			}
		case commMyNick:
			{
				fmt.Printf("\r*** %s has joined to chat ***\n", msg[1])
				fmt.Print("<- ")

				i := strings.Compare(addr.String(), connection.localConn.LocalAddr().String())

				//fmt.Printf("- %d %v %v %v %v\n", i, len(msg[1]), len(name), msg[1] == name, i != 0)

				if (msg[1] == name) && (i != 0) {
					fmt.Println("i have a nick and he is mine")

					//destAddress, err := net.ResolveUDPAddr("udp", addr.String())
					//check(err)

					message := fmt.Sprintf("%s:%s", commNickExist, name)
					//s fmt.Println(messa)
					buffer := make([]byte, len(message))
					copy(buffer, []byte(message))

					//_, err = connection.localConn.WriteToUDP(buffer, addr)
					_, err = connection.localConn.WriteToUDP(buffer, connection.mcastAddress)
					check(err)
					//break
				}
				fmt.Print("\r<- ")
				//break
			}
		case commNickExist:
			{

				i := strings.Compare(addr.String(), connection.localConn.LocalAddr().String())
				if msg[1] == name && (i != 0) {
					//fmt.Print("commLineExists\n")

					newName := "User" + strconv.Itoa(myRand.Intn(1000))
					fmt.Printf("\r*** Nick %s already exists. Changing to %s ***\n", name, newName)
					fmt.Printf("*** To change nick type '/nick NEW_NICKNAME' ***\n")
					fmt.Print("<- ")

					//check(err)
					name = newName
					message := fmt.Sprintf("%s:%s", commMyNick, name)
					//fmt.Println("ip ", message)
					buffer := make([]byte, len(message))
					copy(buffer, []byte(message))
					_, err = connection.localConn.WriteToUDP(buffer, connection.mcastAddress)
					check(err)

					//break
				}

			}
		default:
			{
				fmt.Println("you get strange message; ignore")
				//continue
			}
		}

	}
	ch <- 1
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}
