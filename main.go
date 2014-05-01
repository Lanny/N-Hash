package main

import (
  "fmt"
  "bufio"
  "os"
  "crypto/md5")

const DoD = 40 // Degrees of Deviance 
const CONCURRENCY = 4

type LLNode struct {
  value int
  next *LLNode
}

type Node struct {
  one *Node
  zer *Node
  terminal *LLNode
}

type groupedText struct {
  value string
  group int
}

type mCloseChan struct {
  senders int
  closes int
  wChan chan *groupedText
  lock chan bool
}

func makeMMC(n int) *mCloseChan {
  // Make a new mCloseChan which will need to be closed n times before the
  // wrapped channel is closed. Used for closing channels where there are
  // multiple senders terminating asyncrynously.
  mcc := new(mCloseChan)
  mcc.senders = n
  mcc.wChan = make(chan *groupedText, n)
  mcc.lock = make(chan bool, 1)

  return mcc
}

func mClose(mcc *mCloseChan) {
  mcc.lock <- true
  mcc.closes++

  if (mcc.closes >= mcc.senders) {
    close(mcc.wChan)
  }
  <-mcc.lock
}

func LLLen(curNode *LLNode) int {
  length := 0
  for curNode != nil {
    length++
    curNode = curNode.next
  }

  return length
}

func ExpandTree(root *Node, code []byte, value int) {
  var i, j, k uint
  var bit byte
  var newNode *Node
  curNode := root
  for i = 0; i<256; i++ {
    j = i/8
    k = 7-(i%8)

    bit = (code[j] >> k) & 1;

    if bit == 0 {
      newNode = curNode.zer
      if newNode == nil {
        newNode = new(Node)
        curNode.zer = newNode
      }
    } else {
      newNode = curNode.one
      if newNode == nil {
        newNode = new(Node)
        curNode.one = newNode
      }
    }

    curNode = newNode
  }

  // Create a linked list node and make it the head of list stored at this leaf
  lln := new(LLNode)
  lln.value = value
  lln.next = curNode.terminal

  curNode.terminal = lln
}

func FindNeighbors(root *Node, code []byte, deviance int) []*Node {
  // Given a Nhash b-tree find and return all leaf nodes that have 255-n or
  // more bits in common with the given code where n is deviance
  return rFindNeighbors(root, code, deviance, 0)
}

func rFindNeighbors(curNode *Node, code []byte,
                    deviance int, depth uint) []*Node {
  // Hey, we reached a leaf. Sweet.
  if curNode.terminal != nil {
    r := make([]*Node, 1)
    r[0] = curNode
    return r
  }

  var j, k uint
  j = depth/8
  k = 7-(depth%8)

  bit := (code[j] >> k) & 1

  var (
    freeNode *Node
    deviantNode *Node)
  if bit == 0 {
    freeNode = curNode.zer
    deviantNode = curNode.one
  } else {
    freeNode = curNode.one
    deviantNode = curNode.zer
  }

  if (freeNode == nil && deviantNode == nil) ||
     (freeNode == nil && deviance <= 0) {
    return make([]*Node, 0)
  }

  var leaves []*Node
  if freeNode != nil {
    leaves = rFindNeighbors(freeNode, code, deviance, depth+1)
  }

  if deviance > 0 && deviantNode != nil {
    deviantNeighbors := rFindNeighbors(deviantNode, code, deviance-1, depth+1)
    leaves = append(leaves, deviantNeighbors...)
  }

  return leaves
}

func Relate(root *Node, textChan chan []byte, writeLock chan bool,
    resultChan *mCloseChan, idChan chan int) {
  for line := range textChan {
    nCode := Nhash(line)

    neighbors := FindNeighbors(root, nCode, DoD)

    var thisId int // ID of the group we're assigning this inst to, be it new
                   // or already extant.

    if len(neighbors) == 0 {
      // Gotta add a new branch, lock so we're the only writer, read again to
      // make sure nothing has been added since we checked, and if not write
      writeLock <- true

      neighbors = FindNeighbors(root, nCode, DoD)
      if len(neighbors) == 0 {
        thisId = <-idChan
        ExpandTree(root, nCode, thisId)
      } else {
        thisId = neighbors[0].terminal.value
      }
      <-writeLock
    } else {
      thisId = neighbors[0].terminal.value
    }

    g := new(groupedText)
    g.group = thisId
    g.value = string(line)

    resultChan.wChan <- g
  }

  mClose(resultChan)
}

func generateUniqueIds(comm chan int, terminate chan int) {
  // Takes an unbuffered channel and continually puts sequential ints to it,
  // and returns when first value is given to `terminate`.
  i := 0
  for len(terminate) == 0 {
    comm <- i
    i++
  }
}

func LeafCount(root *Node, countSets bool) int {
  // If countSets is false, return the number of unique leaves in the tree,
  // otherwise return the sum of leaf set counts (unique entries in the tree
  // verses total entries in the tree)
  total := 0

  if root.terminal != nil {
    if countSets {
      return LLLen(root.terminal)
    } else {
      return 1
    }
  }

  if root.one != nil {
    total += LeafCount(root.one, countSets)
  }
  if root.zer != nil {
    total += LeafCount(root.zer, countSets)
  }

  return total
}

func smallHash(arr []byte) int {
  result := md5.Sum(arr);
  return int(result[len(result)-1])
}

func Nhash(s []byte) []byte {
  var counter [256]int
  for i, _ := range counter { counter[i] = 0 }

  for i, _ := range s {
    if i > len(s) - 4 { break }
    idx := smallHash(s[i:i+3])
    counter[idx]++
  }

  sum := 0
  for _, v := range counter { sum += v }
  mean := float32(sum) / 256.

  var code [32]byte
  for i, _ := range code {
    for k := 0; k<8; k++ {
      var orer byte
      if float32(counter[i*8+k]) >= mean {
        orer = 1
      } else {
        orer = 0
      }

      code[i] = (code[i] << 1) | orer
    }
  }

  return code[:]
}

func main() {
  SearchTree := new(Node)

  grouped := make([]*groupedText, 0)

  idChan := make(chan int, 0)
  idGenCloseChan := make(chan int, 1)

  go generateUniqueIds(idChan, idGenCloseChan)

  tweetFile, err := os.Open("tweets.txt")
  reader := bufio.NewReader(tweetFile)

  inChan := make(chan []byte, CONCURRENCY)
  writeLock := make(chan bool, 1)
  outChan := makeMMC(CONCURRENCY)

  for i := 0; i < CONCURRENCY; i++ {
    go Relate(SearchTree, inChan, writeLock, outChan, idChan)
  }

  // Read those suckers in
  go func() {
    var line []byte
    for ; err == nil; line, err = reader.ReadSlice('\n') {
      inChan <- line
    }

    close(inChan)
  }()

  for g := range outChan.wChan {
    grouped = append(grouped, g)
  }

  idGenCloseChan <- 1
  fmt.Println("Last id:", <-idChan)

  fmt.Printf("With %d degrees of deviance, %d entries were grouped into " +
      "%d unique groups\n",
    DoD, len(grouped), LeafCount(SearchTree, false))
}
