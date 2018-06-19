package integration_tests

import (
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration", func() {
	var port string
	var credhubSession *gexec.Session

	BeforeEach(func() {
		cleanupCredhubDatabase()
		port = getFreePort()
		credhubSession = startCredhub(port, []string{"integration-database", "integration-enable-acls"})
	})

	AfterEach(func() {
		gexec.KillAndWait()
	})

	It("adds permissions after restart", func() {
		loginUser("credhub", "password", port)

		session := credhubCommand("set", "-n", "cred", "-t", "password", "-w", "value")
		Eventually(session, "10s").Should(gbytes.Say("/cred"))
		Eventually(session).Should(gexec.Exit(0))

		loginUser("credhub2", "password", port)

		session = credhubCommand("get", "-n", "cred")
		Eventually(session.Err, "10s").Should(gbytes.Say("The request could not be completed because the credential does not exist or you do not have sufficient authorization"))
		Eventually(session).Should(gexec.Exit(1))

		loginClient("credhub_client", "secret", port)

		session = credhubCommand("get", "-n", "cred")
		Eventually(session.Err, "10s").Should(gbytes.Say("The request could not be completed because the credential does not exist or you do not have sufficient authorization"))
		Eventually(session).Should(gexec.Exit(1))

		credhubSession.Kill().Wait()
		port = getFreePort()
		credhubSession = startCredhub(port, []string{"integration-database", "integration-enable-acls", "integration-permissions"})

		loginUser("credhub2", "password", port)

		session = credhubCommand("get", "-n", "cred")
		Eventually(session, "10s").Should(gbytes.Say("/cred"))
		Eventually(session).Should(gexec.Exit(0))

		loginClient("credhub_client", "secret", port)

		session = credhubCommand("get", "-n", "cred")
		Eventually(session, "10s").Should(gbytes.Say("/cred"))
		Eventually(session).Should(gexec.Exit(0))
	})
})

func loginUser(username string, password string, port string) {
	login([]string{"login", "-s", "https://localhost:" + port, "-u", username, "-p", password, "--skip-tls-validation"})
}

func loginClient(clientName string, clientSecret string, port string) {
	login([]string{"login", "-s", "https://localhost:" + port, "--client-name", clientName, "--client-secret", clientSecret, "--skip-tls-validation"})
}

func login(args []string) {
	session := credhubCommand(args...)
	Eventually(session, "10s").Should(gexec.Exit(0))
}

func credhubCommand(args ...string) *gexec.Session {
	cmd := exec.Command("credhub", args...)
	cmd.Env = removeCredhubEnvVars(cmd.Env)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}

func cleanupCredhubDatabase() {
	cmd := exec.Command("psql", "-U", "pivotal", "-c", "DROP DATABASE IF EXISTS credhub_integration", "-c", "CREATE DATABASE credhub_integration")
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
}

func getFreePort() string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	Expect(err).NotTo(HaveOccurred())
	defer listener.Close()
	return strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
}

func startCredhub(port string, additionalProfiles []string) *gexec.Session {
	credhubDir, err := filepath.Abs("../credhub")
	Expect(err).NotTo(HaveOccurred())

	testsDir, err := filepath.Abs(".")
	Expect(err).NotTo(HaveOccurred())

	activeProfiles := "-Dspring.profiles.active=dev"
	if len(additionalProfiles) > 0 {
		activeProfiles += "," + strings.Join(additionalProfiles, ",")
	}
	cmd := exec.Command("./start_server.sh", fmt.Sprintf("-Dspring.config.location=file:%s/fixtures/", testsDir), activeProfiles, "-Dserver.port="+port)
	cmd.Dir = credhubDir
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	timeout := time.After(time.Minute)
	for {
		select {
		case <-timeout:
			Fail("Server did not start after one minute")
		default:
			if _, err := net.Dial("tcp", "127.0.0.1:"+port); err == nil {
				return session
			}
		}
	}
}

func removeCredhubEnvVars(vars []string) []string {
	var newVars []string
	for _, v := range vars {
		if !strings.HasPrefix(v, "CREDHUB") {
			newVars = append(newVars, v)
		}
	}
	return newVars
}
