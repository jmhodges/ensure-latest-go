package main

func updateGitHubActionVersionFile(fpath, oldGoVers, goVers string) ([]fileContent, error) {
	if oldGoVers == goVers {
		return nil, nil
	}
	return []fileContent{
		{
			origFP:          fpath,
			contentsToWrite: []byte(goVers),
		},
	}, nil
}
