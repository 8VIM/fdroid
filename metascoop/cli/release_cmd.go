package cli

type ReleaseCmd struct {
	App     string `arg:"" help:"app" optional:""`
	Version string `arg:"" help:"Release version" optional:""`
}

func (c *ReleaseCmd) Run(g *Globals) (err error) {
	if c.App == "" || c.Version == "" {
		err = g.loader.All(g.RepoDir)
	} else {
		err = g.loader.FromRelease(g.RepoDir, c.App, c.Version)
	}
	if err != nil {
		return
	}
	err = g.updateAndPull()
	return
}
