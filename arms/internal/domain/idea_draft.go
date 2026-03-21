package domain

// IdeaDraft is produced by ideation before persistence assigns IdeaID.
type IdeaDraft struct {
	Title       string
	Description string
	Impact      float64
	Feasibility float64
	Reasoning   string
}
