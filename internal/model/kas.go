package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// What a kas entry is: money coming into a cash book, or going out.
const (
	KasIn  = "in"
	KasOut = "out"
)

// KasEntry is one line of a cash book: a kolom's ("Kas Kolom": Kolom > 0)
// or a BIPRA category's ("Kas BIPRA", for the whole congregation: BipraID
// set, Kolom 0). Cash in (mostly the offering of an ibadah) or cash out;
// amounts are whole rupiah.
type KasEntry struct {
	ID      bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	Kolom   int            `bson:"kolom" json:"kolom"`
	BipraID *bson.ObjectID `bson:"bipraId,omitempty" json:"bipraId"`
	Kind    string         `bson:"kind" json:"kind"`     // KasIn or KasOut
	Date    time.Time      `bson:"date" json:"date"`     // midnight UTC of the local date
	Amount  int64          `bson:"amount" json:"amount"` // rupiah, more than 0
	// ServiceID links the entry to the ibadah the cash came from (or was
	// spent for); nil: another source or use, told by Description.
	ServiceID *bson.ObjectID `bson:"serviceId,omitempty" json:"serviceId"`
	// ServiceLabel names that ibadah as it was when saved ("Ibadah Kolom ·
	// Kel. Umboh · 2026-11-03"), so the entry still says it if the service
	// is deleted later
	ServiceLabel string    `bson:"serviceLabel,omitempty" json:"serviceLabel"`
	Description  string    `bson:"description" json:"description"`
	CreatedBy    string    `bson:"createdBy" json:"createdBy"`
	UpdatedBy    string    `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time `bson:"updatedAt" json:"updatedAt"`
	// Book names the book in answers ("Kolom 3", "BIPRA Pemuda"; not stored)
	Book string `bson:"-" json:"book,omitempty"`
}

// KasSummary is a cash book's totals: a kolom's, or a BIPRA category's
// (BipraID and Name set, Kolom 0).
type KasSummary struct {
	Kolom   int            `bson:"-" json:"kolom"`
	BipraID *bson.ObjectID `bson:"-" json:"bipraId,omitempty"`
	Name    string         `bson:"-" json:"name,omitempty"` // the category's
	Key     any            `bson:"_id" json:"-"`            // what the store grouped by
	In      int64          `bson:"in" json:"in"`
	Out     int64          `bson:"out" json:"out"`
	Count   int            `bson:"count" json:"count"`
	LastDay *time.Time     `bson:"last" json:"last"` // date of the latest entry
	// MonthIn and MonthOut are the totals of the month asked for (see Month)
	MonthIn  int64  `bson:"monthIn" json:"monthIn"`
	MonthOut int64  `bson:"monthOut" json:"monthOut"`
	Month    string `bson:"-" json:"month"` // "2026-11"
}

// KasLabel names an entry for the activity log: "Kolom 3 · masuk Rp 1.250.000 · 2026-11-03".
func KasLabel(book, kind string, amount int64, day string) string {
	what := "masuk"
	if kind == KasOut {
		what = "keluar"
	}
	return fmt.Sprintf("%s · %s %s · %s", book, what, Rupiah(amount), day)
}

// KolomBook names a kolom's cash book: "Kolom 3".
func KolomBook(k int) string { return fmt.Sprintf("Kolom %d", k) }

// BipraBook names a BIPRA category's cash book: "BIPRA Pemuda".
func BipraBook(name string) string { return "BIPRA " + name }

// Rupiah writes an amount the Indonesian way: "Rp 1.250.000".
func Rupiah(amount int64) string {
	sign := ""
	if amount < 0 {
		sign, amount = "-", -amount
	}
	s := strconv.FormatInt(amount, 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return sign + "Rp " + b.String()
}
