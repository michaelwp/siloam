package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// WorshipType is a kind of service ("jenis ibadah"), e.g. Kolom or Pemuda.
type WorshipType struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string        `bson:"name" json:"name"`
	Color     string        `bson:"color" json:"color"` // #rrggbb, the type's calendar color
	Order     int           `bson:"order" json:"order"`
	CreatedAt time.Time     `bson:"createdAt" json:"createdAt,omitzero"`
}

// DefaultWorshipTypes seed a new database, in this order.
var DefaultWorshipTypes = []WorshipType{
	{Name: "Kolom", Color: "#039be5"},
	{Name: "Wanita Kaum Ibu (WKI)", Color: "#e67c73"},
	{Name: "Pria Kaum Bapa (PKB)", Color: "#3f51b5"},
	{Name: "Pemuda", Color: "#f4511e"},
	{Name: "Remaja", Color: "#33b679"},
	{Name: "Anak Sekolah Minggu (ASM)", Color: "#f6bf26"},
}

// Service is one scheduled worship service ("Jadwal Ibadah" row).
type Service struct {
	ID       bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Kolom    int           `bson:"kolom" json:"kolom"`
	TypeID   bson.ObjectID `bson:"typeId" json:"typeId"`
	Date     time.Time     `bson:"date" json:"date"` // midnight UTC of the local date
	Time     string        `bson:"time" json:"time"` // "19:00", local time
	Host     string        `bson:"host" json:"host"` // host family ("Kel. …"), without the prefix
	Location string        `bson:"location" json:"location"`
	Leader   string        `bson:"leader" json:"leader"` // pemimpin ibadah
	MC       string        `bson:"mc" json:"mc"`         // pembawa acara
	Note     string        `bson:"note" json:"note"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt,omitzero"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt,omitzero"`
}

// ServiceDuration is the assumed length of a service (for calendar entries).
const ServiceDuration = 90 * time.Minute

// Start is when the service begins in loc.
func (s Service) Start(loc *time.Location) time.Time {
	var h, m int
	if t, err := time.Parse("15:04", s.Time); err == nil {
		h, m = t.Hour(), t.Minute()
	}
	return time.Date(s.Date.Year(), s.Date.Month(), s.Date.Day(), h, m, 0, 0, loc)
}

// ServiceView is a Service with its type and absolute start/end times.
type ServiceView struct {
	Service
	TypeName  string    `json:"typeName"`
	TypeColor string    `json:"typeColor"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
}

func NewServiceView(s Service, t WorshipType, loc *time.Location) ServiceView {
	start := s.Start(loc)
	return ServiceView{Service: s, TypeName: t.Name, TypeColor: t.Color, Start: start, End: start.Add(ServiceDuration)}
}

// Title is the calendar title, e.g. "Ibadah Kolom · Kel. Umboh Riedel".
func (v ServiceView) Title() string {
	t := "Ibadah " + v.TypeName
	if v.Host != "" {
		t += " · Kel. " + v.Host
	}
	return t
}
