package main

import (
	"net"
	"time"
)

type userInfo struct {
	name      string
	chsender  chan int
	chreciver chan int
}

type connPointers struct {
	localAddress *net.UDPAddr
	mcastAddress *net.UDPAddr
	mcastConn    *net.UDPConn
	localConn    *net.UDPConn
}

type msgStore struct {
	msgBody      string
	answerStatus map[string]bool
	userCount    int
}

type globalData struct {
	user         userInfo
	lastUserPing map[string]time.Time
	sendedMsg    map[time.Time]*msgStore
	conn         connPointers
}

const (
	sysGotMSg     string = "GOT_MSG"
	commGetNicks  string = "GET_USERNAME"
	commMsg       string = "MSG"
	commMyNick    string = "MY_NICK"
	commNickExist string = "NICK_EXIST"
	commPrivate   string = "PRIVATE"
	commExit      string = "QUIT"
	commPing      string = "PING"
	commGotMsg    string = "GOT_IT"
)

const (
	usagePrivate    string = "'/private' command usage: " + userCommPrivate + " NICK MESSAGE"
	usageChangeNick string = "To change nick type '" + userCommChangeNick + " NEW_NICKNAME'"
	usageExit       string = "To exit type '" + userCommExit + "'"
	usageGetNicks   string = "To show list of users, type '" + userCommGetUsers + "'"
)

const (
	tagNewName string = "*new"
)

const (
	userCommChangeNick string = "/nick"
	userCommPrivate    string = "/private"
	userCommExit       string = "/quit"
	userCommGetUsers   string = "/users"
)
