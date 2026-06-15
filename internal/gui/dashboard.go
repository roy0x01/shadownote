package gui

import (
	"fmt"
	"math"
	"time"
)

type dashboardData struct {
	Stats      dashboardStats
	Charts     dashboardCharts
	Recent     []dashboardDoc
	Publishing dashboardPublishing
}

type dashboardStats struct {
	Total, Public, Drafts, Private, Research, Pages, Assets int
}

type dashboardCharts struct {
	ContentState    []donutSlice
	WorkspaceTotals []dashboardBar
}

type donutSlice struct {
	Label     string
	Value     int
	URL       string
	ColorIdx  int
	DashArray string
	DashOff   string
}

type dashboardBar struct {
	Label string
	Value int
	URL   string
	Pct   string
}

type dashboardPublishing struct {
	Target    string
	Theme     string
	SiteState string
	Port      int
}

type dashboardDoc struct {
	Title         string
	Slug          string
	Type          string
	Status        string
	Tags          []string
	ModifiedLabel string
}

func (s *Server) buildDashboard(now time.Time) (dashboardData, error) {
	metas, err := s.svc.AllMeta()
	if err != nil {
		return dashboardData{}, err
	}
	cfg := s.svc.ConfigSnapshot()
	assets, _ := s.svc.ListAssets()

	bucket := func(typ, status string) string {
		switch {
		case typ == "page":
			return "pages"
		case typ == "research":
			return "research"
		case status == "live":
			return "public"
		case status == "private":
			return "private"
		default:
			return "drafts"
		}
	}

	var st dashboardStats
	st.Total = len(metas)
	st.Assets = len(assets)
	for _, m := range metas {
		switch bucket(m.Type, m.Status) {
		case "pages":
			st.Pages++
		case "research":
			st.Research++
		case "public":
			st.Public++
		case "private":
			st.Private++
		default:
			st.Drafts++
		}
	}

	slices := []donutSlice{
		{Label: "posts", Value: st.Public, URL: "/content?type=post", ColorIdx: 0},
		{Label: "drafts", Value: st.Drafts, URL: "/content?type=draft", ColorIdx: 1},
		{Label: "notes", Value: st.Private, URL: "/content?type=note", ColorIdx: 2},
		{Label: "research", Value: st.Research, URL: "/content?type=research", ColorIdx: 3},
		{Label: "pages", Value: st.Pages, URL: "/content?type=page", ColorIdx: 4},
	}
	computeDonut(slices, st.Total)

	totals := []dashboardBar{
		{Label: "posts", Value: st.Public, URL: "/content?type=post"},
		{Label: "drafts", Value: st.Drafts, URL: "/content?type=draft"},
		{Label: "pages", Value: st.Pages, URL: "/content?type=page"},
		{Label: "assets", Value: st.Assets, URL: "/assets"},
	}
	maxTotal := 0
	for _, b := range totals {
		if b.Value > maxTotal {
			maxTotal = b.Value
		}
	}
	for i := range totals {
		totals[i].Pct = pctOf(totals[i].Value, maxTotal)
	}

	recent := make([]dashboardDoc, 0, 5)
	for i, m := range metas {
		if i >= 5 {
			break
		}
		v := metaToView(m, now)
		recent = append(recent, dashboardDoc{Title: v.Title, Slug: v.Slug, Type: v.Type, Status: v.Status, Tags: v.Tags, ModifiedLabel: v.Modified})
	}

	target := cfg.Deployment.Target
	if target == "" {
		target = "localhost"
	}
	port := cfg.Deployment.Localhost.Port
	if port == 0 {
		port = 8081
	}
	siteState := "Ready"
	if target == "archive" {
		siteState = "Ready to export"
	} else if running, _ := s.svc.LocalRunning(); running {
		siteState = "Running locally"
	}

	return dashboardData{
		Stats:  st,
		Charts: dashboardCharts{ContentState: slices, WorkspaceTotals: totals},
		Recent: recent,
		Publishing: dashboardPublishing{
			Target:    target,
			Theme:     cfg.Blog.Theme.Name,
			SiteState: siteState,
			Port:      port,
		},
	}, nil
}

func computeDonut(slices []donutSlice, total int) {
	const c = 2 * math.Pi * 60
	if total <= 0 {
		return
	}
	cum := 0.0
	for i := range slices {
		frac := float64(slices[i].Value) / float64(total)
		arc := frac * c
		slices[i].DashArray = fmt.Sprintf("%.2f %.2f", arc, c-arc)
		slices[i].DashOff = fmt.Sprintf("%.2f", -cum*c)
		cum += frac
	}
}

func pctOf(v, max int) string {
	if max <= 0 || v <= 0 {
		return "0"
	}
	return fmt.Sprintf("%.1f", float64(v)/float64(max)*100)
}
