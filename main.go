package main

import (
  "fmt"
  "bufio"
  "os"
  "crypto/md5")


type LLNode struct {
  value string
  next *LLNode
}

type Node struct {
  one *Node
  zer *Node
  terminal *LLNode
}

func LLLen(curNode *LLNode) int {
  length := 0
  for curNode != nil {
    length++
    curNode = curNode.next
  }

  return length
}

func ExpandTree(root *Node, code []byte, value string) {
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

  tweetFile, err := os.Open("tweets.txt")
  reader := bufio.NewReader(tweetFile)

  var line []byte
  for ; err == nil; line, err = reader.ReadSlice('\n') {
    nCode := Nhash(line)
    ExpandTree(SearchTree, nCode, string(line))
  }

  knownMember := []byte("RT @wilw: RT @Theremina: This hilarious graph of Netflix speeds shows the importance of net neutrality http://t.co/B2yMqAkyuC")
  kmCode := Nhash(knownMember)

  neighbors := FindNeighbors(SearchTree, kmCode, 20)
  fmt.Println(neighbors)

  for _, tN := range neighbors {
    ll := tN.terminal
    fmt.Println("length:", LLLen(ll))

    curNode := ll
    for curNode != nil {
      fmt.Println(curNode.value)
      curNode = curNode.next
    }
  }
}
