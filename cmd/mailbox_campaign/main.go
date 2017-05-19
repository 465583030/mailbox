// mailbox_campaign - create, list, hash campaign info.
/*

Usage:

	mailbox_campaign [flags ...] [cmd]

The cmd are:

	`create`: create a campaign
	`list`: list all campaign info
	`hash`: make a hash with c(cid), u(uid)

The flags are:

	`-c`: cid
	`-u`: uid

*/
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/toomore/mailbox/campaign"
	"github.com/toomore/mailbox/utils"
)

var (
	conn *sql.DB
	cid  = flag.String("cid", "", "campaign id")
	uid  = flag.String("uid", "", "User id")
)

func create() ([8]byte, [8]byte) {
	id, seed := utils.GenSeed(), utils.GenSeed()
	_, err := conn.Query(fmt.Sprintf(`INSERT INTO campaign(id,seed) VALUES('%s', '%s')`, id, seed))
	defer conn.Close()
	if err != nil {
		log.Fatal(err)
	}
	return id, seed
}

func list() {
	rows, err := conn.Query(`SELECT id,seed,created,updated FROM campaign ORDER BY updated DESC`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var (
		id      string
		seed    string
		created time.Time
		updated time.Time
	)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "id", "seed", "created", "updated*")
	for rows.Next() {
		if err := rows.Scan(&id, &seed, &created, &updated); err != nil {
			log.Println("[err]", err)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, seed, created, updated)
		}
	}
	w.Flush()
}

func makeHash() {
	data := url.Values{}
	data.Add("c", *cid)
	data.Add("u", *uid)
	log.Printf("/read/%x?%s\n", campaign.MakeMac(*cid, data), data.Encode())
}

func openGroups(cid string, groups string) {
	rows, err := conn.Query(`
	SELECT id,email,f_name,reader.created
	FROM user
	LEFT JOIN reader ON (id=reader.uid AND reader.cid=?)
	WHERE groups=?
	GROUP BY id;`, cid, groups)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var (
		id        string
		email     string
		fname     string
		created   sql.NullString
		nums      int
		openCount int
	)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "id*", "email", "f_name", "open")
	for rows.Next() {
		if err := rows.Scan(&id, &email, &fname, &created); err != nil {
			log.Println("[err]", err)
		} else {
			nums++
			if created.Valid {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, email, fname, created.String)
				openCount++
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, email, fname, "Not open")
			}
		}
	}
	fmt.Fprintf(w, "%s\t%s\t%s\t%.2f%%\n", "", "", "", float64(openCount)/float64(nums)*100)
	w.Flush()
}

func openList(cid string, groups string) {
	rows, err := conn.Query(`
	SELECT uid,u.email,count(*) AS count, min(reader.created) as open, max(reader.created) as latest
	FROM reader, user AS u
	WHERE uid=u.id AND cid=? AND u.groups=?
	GROUP BY uid
	ORDER BY count DESC`, cid, groups)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var (
		count  int
		email  string
		nums   int
		fopen  string
		latest string
		sum    int
		uid    string
	)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "uid", "email", "count*", "open", "latest")
	for rows.Next() {
		if err := rows.Scan(&uid, &email, &count, &fopen, &latest); err != nil {
			log.Println("[err]", err)
		} else {
			sum += count
			nums++
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", uid, email, count, fopen, latest)
		}
	}
	fmt.Fprintf(w, "%d\t%.02f%%\t%d\n", nums, float64(sum)/float64(nums)*100, sum)
	w.Flush()
}

func openHistory(cid string, groups string) {
	rows, err := conn.Query(`
	SELECT no,uid,u.email,u.f_name,reader.created,ip,agent
	FROM reader, user AS u
	WHERE cid=? AND uid=u.id AND u.groups=?
	ORDER BY reader.created ASC;
	`, cid, groups)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var (
		no      string
		uid     string
		email   string
		fname   string
		created time.Time
		ip      string
		agent   string
		count   int
	)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "no", "uid", "email", "fname", "created*", "ip", "agent")
	for rows.Next() {
		if err := rows.Scan(&no, &uid, &email, &fname, &created, &ip, &agent); err != nil {
			log.Println("[err]", err)
		} else {
			count++
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", no, uid, email, fname, created, ip, agent)
		}
	}
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "no", "uid", "email", "fname", "created*", "ip", "agent")
	w.Flush()
	fmt.Printf("Count: %d\n", count)
}

func printTips() {
	fmt.Println(`mailbox_campaign [cmd]
  cmd:
	create,
	list,
	-c [cid] -u [userID] hash,
	open [cid] [groups],
	openlist [cid] [groups],
	openhistory [cid] [groups]`)
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) >= 1 {
		conn = utils.GetConn()
		switch args[0] {
		case "create":
			id, seed := create()
			log.Printf("id: %s, seed: %s", id, seed)
		case "list":
			list()
		case "hash":
			makeHash()
		case "open":
			if len(args) >= 3 {
				openGroups(args[1], args[2])
			} else {
				printTips()
			}
		case "openlist":
			if len(args) >= 3 {
				openList(args[1], args[2])
			} else {
				printTips()
			}
		case "openhistory":
			if len(args) >= 3 {
				openHistory(args[1], args[2])
			} else {
				printTips()
			}
		default:
			printTips()
		}
	} else {
		printTips()
	}
}
