package engine

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/nuwa/bpp.v3/common"
	"github.com/nuwa/bpp.v3/environment"
	"github.com/pkg/sftp"
	"github.com/samber/lo"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"path"
	"strings"
)

const colonySSHKeyPrefix = "GS_SERVER_"

func ExecuteSSHReleaseService(working, server, serverPath, before, after string) error {
	// 读取目录或者文件
	installPackagePath, ok := environment.Get("P_OUTPUT")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_OUTPUT"))
	}
	// 压缩文件
	err := common.Zip(installPackagePath, path.Join(working, "install.zip"), false)
	if err != nil {
		return err
	}

	color.Blue(fmt.Sprintf("[SSH] 服务器: %s 服务器路径: %s 执行脚本 (before): %s 执行脚本 (after): %s", server, serverPath, before, after))
	// 读取环境变量
	value, ok := environment.Get(colonySSHKeyPrefix + strings.ToUpper(server))
	if !ok {
		return errors.New(fmt.Sprintf("SSH Config %s Find Not", colonySSHKeyPrefix+strings.ToUpper(server)))
	}
	index := strings.Index(value, "#")
	if index <= 0 {
		return errors.New(fmt.Sprintf("SSH Config %s Error", colonySSHKeyPrefix+strings.ToUpper(server)))
	}
	var typeName = value[0:index]
	if strings.ToUpper(typeName) != "USER" {
		return errors.New(fmt.Sprintf("SSH Config %s Not User #", colonySSHKeyPrefix+strings.ToUpper(server)))
	}
	var rows = strings.Split(value[index+1:], "\n")
	if len(rows) <= 3 {
		return errors.New(fmt.Sprintf("SSH Config %s Not User #", colonySSHKeyPrefix+strings.ToUpper(server)))
	}
	var host = strings.TrimSpace(rows[0])
	var port = strings.TrimSpace(rows[1])
	var username = strings.TrimSpace(rows[2])
	var password = strings.TrimSpace(rows[3])
	// SSH
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	sshDial, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", host, port), config)
	if err != nil {
		return err
	}
	defer func(sshDial *ssh.Client) {
		_ = sshDial.Close()
	}(sshDial)

	// SFTP
	sftpClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", host, port), config)
	if err != nil {
		return err
	}
	defer func(sftpClient *ssh.Client) {
		_ = sftpClient.Close()
	}(sftpClient)

	sftpSession, err := sftp.NewClient(sftpClient)
	if err != nil {
		return err
	}
	defer func(sftpSession *sftp.Client) {
		_ = sftpSession.Close()
	}(sftpSession)

	// 执行前置命令
	command := []string{
		"mkdir -p " + serverPath,
		"mv -f " + serverPath + " /tmp/gitlab_" + uuid.New().String(),
		"mkdir -p " + serverPath}
	rCommand := lo.If(before == "", "").Else(before+" && ") + strings.Join(command, " && ")
	output, err := runCommand(sshDial, rCommand)
	color.Blue(strings.Join(command, " && "))
	if err != nil {
		return err
	}
	if string(output) != "" {
		color.Blue("================================================================")
		color.Blue(string(output))
		color.Blue("================================================================")
	}

	// 执行上传文件 (目录) 命令
	localFilePath := path.Join(working, "install.zip")
	remoteFilePath := path.Join(serverPath, "install.zip")
	color.Blue(fmt.Sprintf("sftp: %s >>  %s", localFilePath, remoteFilePath))
	err = func() error {
		var localFile *os.File
		localFile, err = os.Open(localFilePath)
		if err != nil {
			return err
		}
		defer func(localFile *os.File) {
			_ = localFile.Close()
		}(localFile)

		var remoteFile *sftp.File
		remoteFile, err = sftpSession.Create(remoteFilePath)
		if err != nil {
			return err
		}
		defer func(remoteFile *sftp.File) {
			_ = remoteFile.Close()
		}(remoteFile)

		// 将本地文件内容复制到远程文件
		_, err = io.Copy(remoteFile, localFile)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return err
	}

	// 执行后置命令
	command = []string{
		"cd " + serverPath,
		"unzip " + remoteFilePath,
		"mv -f " + remoteFilePath + " /tmp/gitlab_" + uuid.New().String()}
	color.Blue(strings.Join(command, " && "))
	output, err = runCommand(sshDial, strings.Join(command, " && "))
	if err != nil {
		return err
	}
	if string(output) != "" {
		color.Blue("================================================================")
		color.Blue(string(output))
		color.Blue("================================================================")
	}

	// 运行脚本
	if after != "" {
		output, err = runCommand(sshDial, after)
		color.Blue(after)
		if err != nil {
			return err
		}
		if string(output) != "" {
			color.Blue("================================================================")
			color.Blue(string(output))
			color.Blue("================================================================")
		}
	}
	return nil
}

// 在新会话中执行命令
func runCommand(client *ssh.Client, command string) ([]byte, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer func(session *ssh.Session) {
		_ = session.Close()
	}(session)
	return session.CombinedOutput(command)
}
