package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fachebot/evm-grid-bot/internal/config"
	"gopkg.in/yaml.v3"
)

// Configurator 配置管理器
type Configurator struct {
	configDir string
}

// NewConfigurator 创建配置管理器
func NewConfigurator(configDir string) *Configurator {
	return &Configurator{
		configDir: configDir,
	}
}

// CopySampleConfig 复制示例配置文件
func (c *Configurator) CopySampleConfig() error {
	// 尝试多个可能的示例文件路径
	possibleSamplePaths := []string{
		filepath.Join(c.configDir, "etc", "config.yaml.sample"), // 相对于配置目录
		filepath.Join("etc", "config.yaml.sample"),              // 相对于当前工作目录
		"etc/config.yaml.sample",                                 // 相对路径
	}
	
	var src *os.File
	var err error
	
	// 查找存在的示例文件
	for _, path := range possibleSamplePaths {
		src, err = os.Open(path)
		if err == nil {
			break
		}
	}
	
	if src == nil {
		return fmt.Errorf("找不到示例配置文件，请确保 etc/config.yaml.sample 文件存在")
	}
	defer src.Close()
	
	configPath := filepath.Join(c.configDir, "etc", "config.yaml")

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 创建目标文件
	dst, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}
	defer dst.Close()

	// 复制内容
	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("复制配置文件失败: %w", err)
	}

	return nil
}

// LoadConfig 加载配置文件
func (c *Configurator) LoadConfig() (*config.Config, error) {
	configPath := filepath.Join(c.configDir, "etc", "config.yaml")
	return config.LoadFromFile(configPath)
}

// SaveConfig 保存配置文件
func (c *Configurator) SaveConfig(cfg *config.Config) error {
	configPath := filepath.Join(c.configDir, "etc", "config.yaml")

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 序列化为YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// UpdateConfig 更新配置项
func (c *Configurator) UpdateConfig(updates func(*config.Config)) error {
	cfg, err := c.LoadConfig()
	if err != nil {
		return err
	}

	updates(cfg)

	return c.SaveConfig(cfg)
}
