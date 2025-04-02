package main

import (
    "bufio"
    "fmt"
    "io"
    "log"
    "os"
    "os/exec"
    "strings"
)

const PASSPHRASE = "your_passphrase_here"
const PASS_DB_FILENAME = "db.gpg"
const ANS_YES_ALIAS = "yes y"
const ANS_NO_ALIAS = "no not n "
const PRINT_ALIAS = "print pr p"
const PRINTALL_ALIAS = "printall prall pa"
const QUERY_ALIAS = "query qu q"
const ADD_ALIAS = "add a"
const EDIT_ALIAS = "edit ed e"
const EDIT_PARAMS_VARIANTS = "label login password"
const DELETE_ALIAS = "delete del d"
const HELP_ALIAS = "help hlp h"
const HELP_TEXT = `
nzkeys (C) nick zaber

simple CLI program to perform basic management on saved accounts. any account
consists of its label (a name by which an account can be accessed, usually
related to a service this account was made in), login and password. all these
fields are contained as strings so feel free to type anything in them or even
omit them (blank labels are not recommended). the program utilizes GnuPG to
encrypt/decrypt the list of accounts using symmetric key.

usage:
    nzkeys print <label>
    nzkeys printall
    nzkeys query <prompt>
    nzkeys add <label> <login> <password>
    nzkeys edit <label> label|login|password <value>
    nzkeys delete <label>
    nzkeys help
where:
    print <label> - print an account with specified <label>.
    printall - print all accounts. make sure no one watches.
    query <prompt> - will print labels that match with <prompt>
    add <label> <login> <password> - add an account to the database
    edit <label> label|login|password <value> - edit field of an account with
      specified <label>. will replace account's label/login/password with
      <value>.
    delete <label> - delete an account with specified <label>. will ask you to
      confirm this action.
    help - print this help and exit
`

type Account struct {
    label string
    login string
    password string
}

func efat(err error) {
    if err != nil {
        log.Fatal(err)
    }
}

func indexof[T comparable](value T, array []T) int {
    for i, v := range array {
        if value == v {
            return i
        }
    }
    return -1
}

func isin[T comparable](value T, array []T) bool {
    if indexof(value, array) != -1 {
        return true
    }
    return false
}

func checkalias(prompt, alias string) bool {
    return isin(prompt, strings.Split(alias, " "))
}

func (acc Account) show() {
    fmt.Println(acc.label)
    fmt.Println(acc.login)
    fmt.Println(acc.password)
}

func getLabels(accs []Account) []string {
    labels := make([]string, len(accs))
    for i, acc := range accs {
        labels[i] = acc.label
    }
    return labels
}

func byLabel(label string, accs []Account) (int, error) {
    for i, acc := range accs {
        if label == acc.label {
            return i, nil
        }
    }
    return -1, fmt.Errorf("%s was not found in database", label)
}

func getDecryptedDB() io.ReadCloser {
    cmd := exec.Command(
        "gpg",
        "-d",
        "--batch",
        "--yes",
        "--passphrase", PASSPHRASE,
        PASS_DB_FILENAME,
    )
    stdout, err := cmd.StdoutPipe()
    efat(err)
    efat(cmd.Start())
    return stdout
}

func getAccs() []Account {
    pipe := getDecryptedDB()
    raw, err := io.ReadAll(pipe)
    efat(err)
    efat(pipe.Close())
    lines := strings.Split(string(raw), "\n")
    var accsCount int = 0
    for _, line := range lines {
        if len(line) < 3 {
            continue
        }
        if line[:2] == "--" {
            accsCount++
        }
    }
    accs := make([]Account, accsCount, accsCount+1)
    var j int = 0
    for i, line := range lines {
        if len(line) < 3 {
            continue
        }
        if line[:2] == "--" {
            accs[j].label = line
            accs[j].login = lines[i+1]
            accs[j].password = lines[i+2]
            j++
        }
    }
    return accs
}

func parseArgs() (string, string, string, string) {
    if len(os.Args[1:]) == 0 {
        log.Fatal("not enough arguments")
    }
    if checkalias(os.Args[1], PRINTALL_ALIAS) {
        return "printall", "", "", ""
    }
    if checkalias(os.Args[1], HELP_ALIAS) {
        return "help", "", "", ""
    }
    if checkalias(os.Args[1], QUERY_ALIAS) {
        if len(os.Args[1:]) < 2 {
            log.Fatal("not enough arguments")
        }
        return "query", os.Args[2], "", ""
    }
    if checkalias(os.Args[1], PRINT_ALIAS) {
        if len(os.Args[1:]) < 2 {
            log.Fatal("not enough arguments")
        }
        return "print", "--"+os.Args[2], "", ""
    }
    if checkalias(os.Args[1], DELETE_ALIAS) {
        if len(os.Args[1:]) < 2 {
            log.Fatal("not enough arguments")
        }
        return "delete", "--"+os.Args[2], "", ""
    }
    if checkalias(os.Args[1], ADD_ALIAS) {
        if len(os.Args[1:]) < 4 {
            log.Fatal("not enough arguments")
        }
        return "add", "--"+os.Args[2], os.Args[3], os.Args[4]
    }
    if checkalias(os.Args[1], EDIT_ALIAS) {
        if len(os.Args[1:]) < 4 {
            log.Fatal("not enough arguments")
        }
        if !checkalias(os.Args[3], EDIT_PARAMS_VARIANTS) {
            log.Fatal("invalid argument #2, label|login|password expected")
        }
        return "edit", "--"+os.Args[2], os.Args[3], os.Args[4]
    }
    log.Fatal("could not recognize command")
    return "", "", "", ""
}

func vault(accs []Account) {
    var raw string = ""
    for _, acc := range accs {
        raw += acc.label + "\n" + acc.login + "\n" + acc.password + "\n"
    }
    err := os.WriteFile("db.tmp", []byte(raw), 0600)
    efat(err)
    cmd := exec.Command(
        "gpg",
        "-c",
        "--batch",
        "--yes",
        "--passphrase", PASSPHRASE,
        "--output", PASS_DB_FILENAME,
        "db.tmp")
    efat(cmd.Start())
    efat(cmd.Wait())
    efat(os.Remove("db.tmp"))
}

func main() {
    log.SetFlags(0)
    command, arg1, arg2, arg3 := parseArgs()
    if command == "help" {
        fmt.Println(HELP_TEXT)
        return
    }
    accs := getAccs()
    var changes bool = false
    switch command {
    case "printall":
        for _, acc := range accs {
            fmt.Println()
            acc.show()
        }
        fmt.Println()
    case "query":
        for _, label := range getLabels(accs) {
            if strings.Contains(
                strings.ToLower(label),
                strings.ToLower(arg1),
            ) {
                fmt.Println(label)
            }
        }
    case "print":
        i, err := byLabel(arg1, accs)
        efat(err)
        accs[i].show()
    case "delete":
        i, err := byLabel(arg1, accs)
        efat(err)
        reader := bufio.NewReader(os.Stdin)
        var answer string
        fmt.Printf("account '%s' will be deleted.\n", arg1)
        for {
            fmt.Printf("are you sure? (yes/NO): ")
            answer, err = reader.ReadString('\n')
            answer = strings.Replace(answer, "\n", "", -1)
            efat(err)
            if checkalias(answer, ANS_NO_ALIAS) {
                fmt.Println("deletion cancelled")
                return
            }
            if checkalias(answer, ANS_YES_ALIAS) {
                break
            }
        }
        accs = append(accs[:i], accs[i+1:]...)
        fmt.Printf("deleted '%s'\n", arg1)
        changes = true
    case "add":
        accs = append(accs, Account{arg1, arg2, arg3})
        fmt.Println("added new account:")
        accs[len(accs)-1].show()
        changes = true
    case "edit":
        i, err := byLabel(arg1, accs)
        efat(err)
        switch arg2 {
        case "label":
            accs[i].label = "--" + arg3
        case "login":
            accs[i].login = arg3
        case "password":
            accs[i].password = arg3
        default:
            log.Fatal("how did you even get there?")
        }
        fmt.Println("now account looks like this:")
        accs[i].show()
        changes = true
    default:
        log.Fatal("how did you even get there?")
    }
    if changes {
        vault(accs)
    }
}
