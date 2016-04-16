package main

import (
	"bufio"
	"fmt"
	//"log"
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
	userNames = make(map[string]time.Time)
	//userNames []string
)

type connStr struct {
	localAddress *net.UDPAddr
	mcastAddress *net.UDPAddr
	mcastConn    *net.UDPConn
	localConn    *net.UDPConn
}

const (
	commGetNicks  string = "GET_USERNAME"
	commMsg       string = "MSG"
	commMyNick    string = "MY_NICK"
	commNickExist string = "NICK_EXIST"
	//commLeave     string = "LEAVED"
	// commChangeNick string = "/NICK"
	commPrivate string = "PRIVATE"
	commExit    string = "QUIT"
	commPing    string = "PING"
)

const (
	usagePrivate    string = "'/private' command usage: " + userCommPrivate + " NICK MESSAGE"
	usageChangeNick string = "To change nick type '" + userCommChangeNick + " NEW_NICKNAME'"
	usageExit       string = "To exit type '" + userCommExit + "'"
	usageGetNicks   string = "To show list of users, type '" + userCommgetUsers + "'"
)

const (
	tagNewName string = "*new"
)

const (
	userCommChangeNick string = "/nick"
	userCommPrivate    string = "/private"
	userCommExit       string = "/quit"
	userCommgetUsers   string = "/users"
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

func usage() {
	fmt.Println("****************************************")
	fmt.Println(usagePrivate)
	fmt.Println(usageChangeNick)
	fmt.Println(usageExit)
	fmt.Println(usageGetNicks)
	fmt.Println("****************************************")
}

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

	message := fmt.Sprintf("%s:%s:%s", commMyNick, tagNewName, name)
	buffer := make([]byte, len(message))
	copy(buffer, []byte(message))
	_, err = connection.localConn.WriteToUDP(buffer, connection.mcastAddress)
	check(err)

	go sender(chsender, connection.localConn, connection.mcastAddress)
	go receiver(chreciver, connection.mcastConn)
	go sendGetPing(chsender, connection.localConn, connection.mcastAddress)
	<-chsender
	<-chreciver
}

func sendGetPing(ch chan int, conn *net.UDPConn, addr *net.UDPAddr) {
	for {

		timer := time.NewTimer(time.Second * 5)
		<-timer.C

		msg := commPing + ":" + name

		buffer := make([]byte, len(msg))
		copy(buffer, []byte(msg))
		_, err := conn.WriteToUDP(buffer, addr)
		check(err)

		for user, lastPing := range userNames {
			diff := time.Now().Sub(lastPing)
			if diff.Seconds() > 5 && user != name {
				fmt.Printf("\r*** %s leaved the chat  ***\n", user)
				delete(userNames, user)
				fmt.Print("<- ")
			}
		}
	}
}

func sender(ch chan int, conn *net.UDPConn, addr *net.UDPAddr) {

	//распределенный консенсунс
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := fmt.Sprintf("%s", scanner.Text())
		fmt.Print("\r")

		var msgParted []string
		msgParted = strings.Split(msg, " ")
		command := msgParted[0]
		switch command {
		case userCommChangeNick:
			{
				if len(msgParted) < 2 || msgParted[1] == name {
					continue
				}
				msg = commMyNick + ":" + name + ":" + msgParted[1]
				name = msgParted[1]

			}
		case userCommPrivate:
			{
				if len(msgParted) < 3 {
					//fmt.Printf("\rSYSTEM: '/private' command usage: %s NICK MESSAGE\n", userCommPrivate)
					fmt.Printf("\r%s\n", usagePrivate)
					fmt.Print("<- ")
					continue
				}
				rawMsg := msg[(len(commPrivate) + len(msgParted[1]) + 3):]
				msg = commPrivate + ":" + msgParted[1] + ":" + name + ": " + rawMsg
				//fmt.Println("row private mes:",rowMsg, "/len: ", len(rowMsg))
				//fmt.Println(msg)
			}
		case userCommExit:
			{
				msg = commExit + ":" + name
				fmt.Println("*** Bye ***")
				//ch <- 1
				buffer := make([]byte, len(msg))
				copy(buffer, []byte(msg))
				_, err := conn.WriteToUDP(buffer, addr)
				check(err)
				os.Exit(0)
			}
		case userCommgetUsers:
			{
				fmt.Println("\rUsers:")
				for user, lastPing := range userNames {
					diff := time.Now().Sub(lastPing)
					if diff.Seconds() < 5 {
						fmt.Print(user, "\t")
					} else {
						fmt.Println(diff.Seconds())
						delete(userNames, "user")
					}
				}
				fmt.Print("\n<- ")
				continue
			}
		default: //just message
			{
				//fmt.Println(msg[:1])
				if msg[:1] == "/" {
					fmt.Println("\rCommand not found =/")
					fmt.Print("<- ")
					continue
				}
				msg = commMsg + ":" + name + ":" + msg
				//fmt.Printf("%s\n", msg)
			}
		}

		buffer := make([]byte, len(msg))
		copy(buffer, []byte(msg))
		_, err := conn.WriteToUDP(buffer, addr)
		check(err)
		fmt.Print("<- ")
	}
	//fmt.Print("Cntl-D for exit. Bye\n")
	os.Exit(0)
	//ch <- 1
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
				//fmt.Println("\rrawMsg ", rawMsg)
				//fmt.Printf("\n%s/%d\n", msg[2][1:], len(msg[2]))
				//fmt.Println("")
				fmt.Print("<- ")

				//fmt.Printf("- %d %v %v %v %v\n", i, len(msg[1]), len(name), msg[1] == name, i != 0)

				var who string
				i := strings.Compare(addr.String(), connection.localConn.LocalAddr().String())
				if i != 0 {
					who = msg[2]
					if msg[2] == name { //names from different ip adds are equal!
						//fmt.Println("i have a nick and he is mine")

						message := fmt.Sprintf("%s:%s", commNickExist, name)
						//s fmt.Println(messa)
						buffer := make([]byte, len(message))
						copy(buffer, []byte(message))

						//_, err = connection.localConn.WriteToUDP(buffer, addr)
						_, err = connection.localConn.WriteToUDP(buffer, connection.mcastAddress)
						check(err)
						//break
					} else { //nick is ok, adding it to userNicks
						//fmt.Println("jj")
						userNames[msg[2]] = time.Now()
					}
					delete(userNames, msg[1])
				} else {
					who = "You"
					userNames[name] = time.Now()
				}

				if msg[1] == tagNewName {
					fmt.Printf("\r*** %s has joined to chat ***\n", who)
					if i == 0 {
						userNames[name] = time.Now()
						usage()
					}
				} else {
					if i == 0 {
						fmt.Printf("\r*** %s changed name to %s ***\n", who, msg[2])
					} else {
						fmt.Printf("\r*** %s changed name to %s ***\n", msg[1], msg[2])
					}
				}

				fmt.Print("\r<- ")
				//break
			}
		case commNickExist:
			{
				i := strings.Compare(addr.String(), connection.localConn.LocalAddr().String())
				if msg[1] == name && (i != 0) { //nick is the same, ip addr is not
					//fmt.Print("commLineExists\n")
					delete(userNames, name)
					newName := "User" + strconv.Itoa(myRand.Intn(1000))
					fmt.Printf("\rSYSTEM: Nick %s already exists. Changing to %s\n", name, newName)
					fmt.Printf("SYSTEM: %s\n", usageChangeNick)
					fmt.Print("<- ")

					//check(err)

					message := fmt.Sprintf("%s:%s:%s", commMyNick, name, newName)

					name = newName
					//fmt.Println("ip ", message)
					buffer := make([]byte, len(message))
					copy(buffer, []byte(message))
					_, err = connection.localConn.WriteToUDP(buffer, connection.mcastAddress)
					check(err)

					//break
				}
			}
		case commPrivate:
			{
				i := strings.Compare(addr.String(), connection.localConn.LocalAddr().String())
				if i != 0 {
					if len(msg) > 2 && msg[1] == name {
						rawMsg = rawMsg[len(msg[0])+len(msg[1])+2:]
						fmt.Printf("\r->[%s] %s\n", msg[0], rawMsg)
						fmt.Print("<- ")
					}
				}
			}
		case commExit:
			{
				fmt.Printf("\r*** %s leaved the chat  ***\n", msg[1])
				delete(userNames, msg[1])
				fmt.Print("<- ")
			}
		case commPing:
			{
				userNames[msg[1]] = time.Now()
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
