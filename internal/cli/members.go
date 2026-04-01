package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/mucan54/envs3/internal/crypto"
	"github.com/spf13/cobra"
)

var (
	membersAddEnv   string
	membersAddEmail string
	membersAddRole  string
	membersRemoveEnv string
	membersRemoveYes bool
)

func init() {
	membersAddCmd.Flags().StringVar(&membersAddEnv, "env", "", "Comma-separated environments to grant access")
	membersAddCmd.Flags().StringVar(&membersAddEmail, "email", "", "Override email")
	membersAddCmd.Flags().StringVar(&membersAddRole, "role", "member", "Role: admin or member")
	membersAddCmd.MarkFlagRequired("env")

	membersRemoveCmd.Flags().StringVar(&membersRemoveEnv, "env", "", "Comma-separated environments to revoke")
	membersRemoveCmd.Flags().BoolVar(&membersRemoveYes, "yes", false, "Skip confirmation")

	membersCmd.AddCommand(membersAddCmd)
	membersCmd.AddCommand(membersRemoveCmd)
	membersCmd.AddCommand(membersListCmd)
	rootCmd.AddCommand(membersCmd)
}

var membersCmd = &cobra.Command{
	Use:   "members",
	Short: "Manage project members",
}

var membersAddCmd = &cobra.Command{
	Use:   "add <pubkey_file>",
	Short: "Add a member to the project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, _, priv, pub, err := loadWriteContext()
		if err != nil {
			return err
		}

		// Read pubkey file
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("read pubkey file: %w", err)
		}

		// Parse "envs3-pubkey-v1 <base64> <email>"
		parts := strings.Fields(strings.TrimSpace(string(data)))
		if len(parts) < 2 || parts[0] != "envs3-pubkey-v1" {
			return fmt.Errorf("invalid pubkey file format. Expected: envs3-pubkey-v1 <base64> <email>")
		}

		newPub, err := crypto.DecodeKey(parts[1])
		if err != nil {
			return fmt.Errorf("decode public key: %w", err)
		}

		email := membersAddEmail
		if email == "" && len(parts) >= 3 {
			email = parts[2]
		}
		if email == "" {
			email = prompt("Email for new member: ")
		}

		envs := strings.Split(membersAddEnv, ",")
		for i := range envs {
			envs[i] = strings.TrimSpace(envs[i])
		}

		if err := eng.AddMember(ctx(), newPub, email, membersAddRole, envs, priv, pub); err != nil {
			return err
		}

		fp := crypto.Fingerprint(newPub)
		fmt.Printf("✓ Added %s (%s) to %s\n", email, fp, strings.Join(envs, ", "))
		return nil
	},
}

var membersRemoveCmd = &cobra.Command{
	Use:   "remove <email>",
	Short: "Remove a member from the project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, _, priv, pub, err := loadWriteContext()
		if err != nil {
			return err
		}

		email := args[0]

		var envs []string
		if membersRemoveEnv != "" {
			for _, e := range strings.Split(membersRemoveEnv, ",") {
				envs = append(envs, strings.TrimSpace(e))
			}
		}

		if !membersRemoveYes {
			if len(envs) == 0 {
				if !confirm(fmt.Sprintf("Remove %s from all environments and the project?", email)) {
					fmt.Println("Cancelled")
					return nil
				}
			} else {
				if !confirm(fmt.Sprintf("Remove %s from %s?", email, strings.Join(envs, ", "))) {
					fmt.Println("Cancelled")
					return nil
				}
			}
		}

		adminEmail := getUserEmail()

		if err := eng.RemoveMember(ctx(), email, envs, priv, pub, adminEmail); err != nil {
			return err
		}

		fmt.Printf("✓ Removed %s\n", email)
		if len(envs) > 0 {
			fmt.Printf("  DEK rotated for: %s\n", strings.Join(envs, ", "))
		}
		return nil
	},
}

var membersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all members",
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, _, _, _, err := loadContext()
		if err != nil {
			return err
		}

		members, err := eng.ListMembers(ctx())
		if err != nil {
			return err
		}

		fmt.Printf("%-25s %-8s %-35s %s\n", "Email", "Role", "Environments", "Added")
		for _, m := range members {
			fmt.Printf("%-25s %-8s %-35s %s\n",
				m.Email, m.Role, strings.Join(m.Environments, ", "), m.AddedAt)
		}
		return nil
	},
}
