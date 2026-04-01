package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/mucan54/envs3/internal/config"
	"github.com/mucan54/envs3/internal/crypto"
	"github.com/mucan54/envs3/internal/engine"
	"github.com/mucan54/envs3/internal/format"
	"github.com/mucan54/envs3/internal/storage"
	"github.com/spf13/cobra"
)

var projectNameRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up envs3 — create a new project or join an existing one",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("envs3 — Setup")
	fmt.Println()

	// --- Step 1: Storage backend & credentials ---

	fmt.Println("Storage backend:")
	fmt.Println("  1. Cloudflare R2")
	fmt.Println("  2. AWS S3")
	fmt.Println("  3. MinIO")
	fmt.Println("  4. Custom S3-compatible")
	choice := prompt("Select (1-4): ")

	var endpoint, region string
	switch choice {
	case "1":
		endpoint = prompt("R2 endpoint URL: ")
		region = "auto"
	case "2":
		region = prompt("AWS region (e.g., us-east-1): ")
		endpoint = fmt.Sprintf("https://s3.%s.amazonaws.com", region)
	case "3":
		endpoint = prompt("MinIO endpoint URL: ")
		region = "us-east-1"
	default:
		endpoint = prompt("S3 endpoint URL: ")
		region = prompt("Region (default: auto): ")
		if region == "" {
			region = "auto"
		}
	}

	bucket := prompt("Bucket name: ")

	fmt.Println()
	fmt.Println("S3 credentials:")
	accessKeyID := prompt("  Access Key ID: ")
	secretAccessKey := prompt("  Secret Access Key: ")

	// --- Step 2: Connect to bucket and discover projects ---

	store := storage.NewS3Store(storage.S3Config{
		Endpoint:        endpoint,
		Region:          region,
		Bucket:          bucket,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	})

	fmt.Print("\nConnecting to bucket... ")
	keys, err := store.List(ctx(), "")
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("cannot connect to storage: %w", err)
	}
	fmt.Println("✓")

	// Find project directories
	type projectInfo struct {
		Name string
		Envs int
	}
	var projects []projectInfo
	seen := make(map[string]bool)

	for _, k := range keys {
		if idx := strings.IndexByte(k, '/'); idx > 0 {
			name := k[:idx]
			if !seen[name] {
				seen[name] = true
				var pf format.ProjectFile
				if data, _, err := store.Get(ctx(), storage.ProjectPath(name)); err == nil {
					if json.Unmarshal(data, &pf) == nil {
						projects = append(projects, projectInfo{Name: name, Envs: len(pf.Environments)})
					}
				}
			}
		}
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	// --- Step 3: Pick existing project or create new ---

	var projectName string
	createNew := false

	if len(projects) > 0 {
		fmt.Println()
		fmt.Println("Projects found in this bucket:")
		for i, p := range projects {
			envLabel := "environment"
			if p.Envs != 1 {
				envLabel = "environments"
			}
			fmt.Printf("  %d. %s (%d %s)\n", i+1, p.Name, p.Envs, envLabel)
		}
		fmt.Printf("  %d. [Create new project]\n", len(projects)+1)

		sel := prompt(fmt.Sprintf("Select (1-%d): ", len(projects)+1))
		idx, _ := strconv.Atoi(sel)
		if idx >= 1 && idx <= len(projects) {
			projectName = projects[idx-1].Name
		} else {
			createNew = true
		}
	} else {
		fmt.Println("\nNo projects found in this bucket.")
		createNew = true
	}

	if createNew {
		return initNewProject(store, endpoint, region, bucket, accessKeyID, secretAccessKey)
	}
	return initExistingProject(store, projectName, endpoint, region, bucket, accessKeyID, secretAccessKey)
}

// initExistingProject joins an existing project
func initExistingProject(store *storage.S3Store, projectName, endpoint, region, bucket, accessKeyID, secretAccessKey string) error {
	fmt.Printf("\n── Connecting to '%s' ──\n\n", projectName)

	var projectFile format.ProjectFile
	data, _, err := store.Get(ctx(), storage.ProjectPath(projectName))
	if err != nil {
		return fmt.Errorf("project '%s' not found: %w", projectName, err)
	}
	if err := json.Unmarshal(data, &projectFile); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}

	fmt.Println("Environments:")
	for _, env := range projectFile.Environments {
		fmt.Printf("  - %s\n", env)
	}

	// Generate keypair
	pub, priv, err := crypto.GenerateKeypair()
	if err != nil {
		return fmt.Errorf("generate keypair: %w", err)
	}
	if err := config.SavePrivateKey(projectName, priv); err != nil {
		return fmt.Errorf("save private key: %w", err)
	}

	defaultEnv := "local"
	if len(projectFile.Environments) > 0 {
		defaultEnv = projectFile.Environments[0]
	}

	// Write .env.envs3 with everything
	envCfg := &format.Envs3Config{
		Project:         projectName,
		Endpoint:        endpoint,
		Bucket:          bucket,
		Region:          region,
		DefaultEnv:      defaultEnv,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}
	if err := format.SaveEnvs3File(".", envCfg, false); err != nil {
		return fmt.Errorf("save .env.envs3: %w", err)
	}

	// Save local state
	state := &format.LocalState{
		Environments:      make(map[string]format.EnvState),
		ActiveEnvironment: defaultEnv,
	}
	config.SaveLocalState(projectName, state)

	fp := crypto.Fingerprint(pub)
	fmt.Println()
	fmt.Printf("✓ Keypair generated → %s\n", config.PrivateKeyPath(projectName))
	fmt.Println("✓ .env.envs3 created")
	fmt.Printf("  Fingerprint: %s\n", fp)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Send your public key to the admin:")
	fmt.Println("     envs3 pubkey --output my.pub")
	fmt.Println("  2. Once the admin adds you, run:")
	fmt.Println("     envs3 pull")
	fmt.Println()
	fmt.Println("Tip: Run 'envs3 json --keep-s3' to split config into")
	fmt.Println("     .envs3.json (committable) + .env.envs3 (secrets only)")

	return nil
}

// initNewProject creates a new project in the bucket
func initNewProject(store *storage.S3Store, endpoint, region, bucket, accessKeyID, secretAccessKey string) error {
	fmt.Println()
	fmt.Println("── Creating new project ──")
	fmt.Println()

	projectName := prompt("Project name: ")
	if !projectNameRegex.MatchString(projectName) {
		return fmt.Errorf("project name must match [a-z0-9-]+")
	}

	email := getUserEmail()

	envsInput := prompt("Environments (comma-separated, default: local,staging,production): ")
	if envsInput == "" {
		envsInput = "local,staging,production"
	}
	envs := strings.Split(envsInput, ",")
	for i := range envs {
		envs[i] = strings.TrimSpace(envs[i])
	}

	// Check write access
	writeKeyID := accessKeyID
	writeSecretKey := secretAccessKey

	fmt.Print("Checking write access... ")
	testKey := projectName + "/.envs3-test"
	_, writeErr := store.Put(ctx(), testKey, []byte("test"), "")
	if writeErr != nil {
		fmt.Println("✗ (read-only)")
		fmt.Println()
		fmt.Println("Your current credentials are read-only. Creating a project requires read-write access.")
		fmt.Println()
		fmt.Println("Read-write S3 credentials:")
		writeKeyID = prompt("  Access Key ID: ")
		writeSecretKey = prompt("  Secret Access Key: ")

		store = storage.NewS3Store(storage.S3Config{
			Endpoint:        endpoint,
			Region:          region,
			Bucket:          bucket,
			AccessKeyID:     writeKeyID,
			SecretAccessKey: writeSecretKey,
		})
	} else {
		fmt.Println("✓")
		store.Delete(ctx(), testKey)
	}

	// Generate keypair
	pub, priv, err := crypto.GenerateKeypair()
	if err != nil {
		return fmt.Errorf("generate keypair: %w", err)
	}
	if err := config.SavePrivateKey(projectName, priv); err != nil {
		return fmt.Errorf("save private key: %w", err)
	}

	// Check for .env import
	var importEnv string
	var importData map[string]string
	if _, err := os.Stat(".env"); err == nil {
		if confirm("Import existing .env?") {
			importEnv = prompt("Import into environment (default: local): ")
			if importEnv == "" {
				importEnv = "local"
			}
			f, err := os.Open(".env")
			if err != nil {
				return err
			}
			defer f.Close()
			importData, err = format.ParseDotEnv(f)
			if err != nil {
				return fmt.Errorf("parse .env: %w", err)
			}
		}
	}

	// Create project in S3
	eng := engine.NewEngine(store, projectName)
	if err := eng.Init(ctx(), engine.InitParams{
		ProjectName:  projectName,
		Email:        email,
		Environments: envs,
		ImportEnv:    importEnv,
		ImportData:   importData,
		PublicKey:    pub,
	}); err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	fmt.Printf("✓ Project created → s3://%s/%s/\n", bucket, projectName)
	if importData != nil {
		fmt.Printf("✓ Environment '%s' created (%d keys imported)\n", importEnv, len(importData))
	}

	// Determine read-only vs write credentials
	readKeyID := accessKeyID
	readSecretKey := secretAccessKey
	if writeKeyID == accessKeyID {
		// Same credentials — ask if they have separate read-only ones
		if confirm("Do you have separate read-only S3 credentials for team members?") {
			readKeyID = prompt("  Read-only Access Key ID: ")
			readSecretKey = prompt("  Read-only Secret Access Key: ")
		}
	}

	// Write .env.envs3 with everything
	envCfg := &format.Envs3Config{
		Project:              projectName,
		Endpoint:             endpoint,
		Bucket:               bucket,
		Region:               region,
		DefaultEnv:           envs[0],
		AccessKeyID:          readKeyID,
		SecretAccessKey:      readSecretKey,
		WriteAccessKeyID:     writeKeyID,
		WriteSecretAccessKey: writeSecretKey,
	}
	if err := format.SaveEnvs3File(".", envCfg, false); err != nil {
		return fmt.Errorf("save .env.envs3: %w", err)
	}

	// Save local state
	state := &format.LocalState{
		Environments:      make(map[string]format.EnvState),
		ActiveEnvironment: envs[0],
	}
	config.SaveLocalState(projectName, state)

	fmt.Printf("✓ Keypair generated → %s\n", config.PrivateKeyPath(projectName))
	fmt.Println("✓ .env.envs3 created")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Share .env.envs3 with your team (remove write credentials first)")
	fmt.Println("  2. Run 'envs3 pull' to sync")
	fmt.Println()
	fmt.Println("Tip: Run 'envs3 json --keep-s3' to split config into")
	fmt.Println("     .envs3.json (committable) + .env.envs3 (secrets only)")

	return nil
}
