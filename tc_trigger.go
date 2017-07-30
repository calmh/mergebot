package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const tcBuildTemplate = `<build branchName="%s">
	<buildType id="%s"/>
	<comment><text>Triggered by mergebot</text></comment>
</build>`

var (
	// export TEAMCITY_SERVER=build2.syncthing.net
	// export TEAMCITY_USER=machine
	// export TEAMCITY_PASSWORD=p4ssw0rd
	// export TEAMCITY_BUILD_IDS=Syncthing_BuildLinuxCross,Syncthing_BuildMac,Syncthing_BuildWindows,Syncthing_CheckAuthors,Syncthing_CheckCorrectness
	tcServer   = os.Getenv("TEAMCITY_SERVER")
	tcUser     = os.Getenv("TEAMCITY_USER")
	tcPassword = os.Getenv("TEAMCITY_PASSWORD")
	tcBuildIDs = strings.Split(os.Getenv("TEAMCITY_BUILD_IDS"), ",")
)

func tcTriggerBuild(pr int) error {
	branch := fmt.Sprintf("pull/%d", pr)
	var firstError error
	for _, buildID := range tcBuildIDs {
		data := fmt.Sprintf(tcBuildTemplate, branch, buildID)
		if err := tcPostBuildRequest(data); err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}

func tcPostBuildRequest(data string) error {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s/httpAuth/app/rest/buildQueue", tcServer), strings.NewReader(data))
	if err != nil {
		return err
	}

	req.SetBasicAuth(tcUser, tcPassword)
	req.Header.Set("Content-Type", "application/xml")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	return nil
}
