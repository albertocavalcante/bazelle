package jvm

import (
	"path/filepath"
	"testing"
)

func TestLanguageFileExtensions(t *testing.T) {
	tests := []struct {
		lang Language
		want []string
	}{
		{Kotlin, []string{".kt", ".kts"}},
		{Groovy, []string{".groovy", ".gvy", ".gy", ".gsh"}},
		{Java, []string{".java"}},
		{Scala, []string{".scala", ".sc"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			got := tt.lang.FileExtensions()
			if len(got) != len(tt.want) {
				t.Errorf("FileExtensions() = %v, want %v", got, tt.want)
				return
			}
			for i, ext := range got {
				if ext != tt.want[i] {
					t.Errorf("FileExtensions()[%d] = %v, want %v", i, ext, tt.want[i])
				}
			}
		})
	}
}

func TestLanguageMainSourceDir(t *testing.T) {
	tests := []struct {
		lang Language
		want string
	}{
		{Kotlin, filepath.Join("src", "main", "kotlin")},
		{Groovy, filepath.Join("src", "main", "groovy")},
		{Java, filepath.Join("src", "main", "java")},
		{Scala, filepath.Join("src", "main", "scala")},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			got := tt.lang.MainSourceDir()
			if got != tt.want {
				t.Errorf("MainSourceDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLanguageTestSourceDir(t *testing.T) {
	tests := []struct {
		lang Language
		want string
	}{
		{Kotlin, filepath.Join("src", "test", "kotlin")},
		{Groovy, filepath.Join("src", "test", "groovy")},
		{Java, filepath.Join("src", "test", "java")},
		{Scala, filepath.Join("src", "test", "scala")},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			got := tt.lang.TestSourceDir()
			if got != tt.want {
				t.Errorf("TestSourceDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLanguageDirectivePrefix(t *testing.T) {
	tests := []struct {
		lang Language
		want string
	}{
		{Kotlin, "kotlin"},
		{Groovy, "groovy"},
		{Java, "java"},
		{Scala, "scala"},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			got := tt.lang.DirectivePrefix()
			if got != tt.want {
				t.Errorf("DirectivePrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLanguageGlobPatterns(t *testing.T) {
	tests := []struct {
		lang   Language
		subDir string
		want   []string
	}{
		{
			Kotlin,
			"src/main/kotlin",
			[]string{
				filepath.Join("src/main/kotlin", "**", "*.kt"),
				filepath.Join("src/main/kotlin", "**", "*.kts"),
			},
		},
		{
			Groovy,
			"src/test/groovy",
			[]string{
				filepath.Join("src/test/groovy", "**", "*.groovy"),
				filepath.Join("src/test/groovy", "**", "*.gvy"),
				filepath.Join("src/test/groovy", "**", "*.gy"),
				filepath.Join("src/test/groovy", "**", "*.gsh"),
			},
		},
		{
			Scala,
			"src/main/scala",
			[]string{
				filepath.Join("src/main/scala", "**", "*.scala"),
				filepath.Join("src/main/scala", "**", "*.sc"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang)+"/"+tt.subDir, func(t *testing.T) {
			got := tt.lang.GlobPatterns(tt.subDir)
			if len(got) != len(tt.want) {
				t.Errorf("GlobPatterns() = %v, want %v", got, tt.want)
				return
			}
			for i, p := range got {
				if p != tt.want[i] {
					t.Errorf("GlobPatterns()[%d] = %v, want %v", i, p, tt.want[i])
				}
			}
		})
	}
}

func TestAllLanguages(t *testing.T) {
	langs := AllLanguages()
	if len(langs) != 4 {
		t.Errorf("AllLanguages() returned %d languages, want 4", len(langs))
	}

	expected := map[Language]bool{
		Kotlin: true,
		Groovy: true,
		Java:   true,
		Scala:  true,
	}

	for _, lang := range langs {
		if !expected[lang] {
			t.Errorf("AllLanguages() contains unexpected language: %v", lang)
		}
	}
}
