package provenance

import "time"

// Gather produces a full provenance Report. townRoot and sourceRoot are
// passed explicitly rather than resolved internally, so callers (lt doctor,
// lt provenance, tests) control what's being inspected.
func Gather(townRoot, sourceRoot string, allRoles []string) *Report {
	r := &Report{
		GeneratedAt: time.Now(),
		TownRoot:    townRoot,
		SourceRoot:  sourceRoot,
	}

	r.Binaries = GatherBinaries()
	r.Shadows = GatherShadows(r.Binaries)

	var ltCommit, ltRealPath string
	for _, b := range r.Binaries {
		if b.Name == "lt" {
			ltCommit = b.Commit
			ltRealPath = b.RealPath
		}
	}
	r.Source = GatherSource(sourceRoot, ltCommit)
	r.Daemon = GatherDaemon(ltRealPath)
	r.Launchd = GatherLaunchd()
	r.Directives = GatherDirectives(townRoot, allRoles)
	r.Roster = GatherRoster()

	return r
}

// Violations returns every provenance finding that should fail `lt doctor`
// loudly (a StatusError), each as a short, standalone message.
func (r *Report) Violations() []string {
	var v []string

	for _, b := range r.Binaries {
		if b.LookupError != "" {
			v = append(v, "binary "+b.Name+" not found on PATH: "+b.LookupError)
		}
	}

	for _, s := range r.Shadows {
		if !s.MatchesCanonical {
			v = append(v, "shadow binary: "+s.Path+" is a different executable than the canonical "+s.Name+" — two products contending for the same name")
		}
	}

	if r.Source.Error != "" {
		v = append(v, "source provenance: "+r.Source.Error)
	} else if r.Source.Relationship == "diverged" {
		v = append(v, "running binary's commit has diverged from "+r.SourceRoot+" HEAD")
	}

	if !r.Daemon.Running {
		v = append(v, "daemon is not running")
	} else if !r.Daemon.MatchesPATHBinary && r.Daemon.ExecutablePath != "" {
		v = append(v, "running daemon executable ("+r.Daemon.ExecutablePath+") does not match the canonical lt on PATH")
	}

	for _, j := range r.Launchd {
		if j.StatusError != "" {
			v = append(v, "launchd job "+j.Label+": "+j.StatusError)
			continue
		}
		if !j.ProgramExists {
			v = append(v, "launchd job "+j.Label+" declares a program that does not exist: "+j.DeclaredProgram)
		}
	}

	for _, d := range r.Directives {
		if !d.DirectiveLess && !d.Present {
			v = append(v, "role "+d.Role+" has no directive file (expected "+d.FileName+".md)")
		}
	}

	return v
}

// Warnings returns findings worth surfacing but not failing the check on
// (e.g. a working tree that's dirty, or a stale roster file that hasn't
// been fully retired yet).
func (r *Report) Warnings() []string {
	var w []string

	if r.Source.Error == "" && !r.Source.WorkingTreeClean {
		w = append(w, "source working tree at "+r.SourceRoot+" is not clean")
	}

	for _, j := range r.Launchd {
		if j.ProgramExists && !j.Loaded {
			w = append(w, "launchd job "+j.Label+" is not currently loaded (won't survive login/boot until loaded)")
		}
	}

	for _, rf := range r.Roster {
		w = append(w, "stale roster file: "+rf.Path+" ("+rf.Reason+")")
	}

	return w
}
