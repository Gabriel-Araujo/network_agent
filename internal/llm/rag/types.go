package rag

// Section representa uma seção do documento RST, com o caminho de
// breadcrumbs (Path) até a raiz.
type Section struct {
	Level     int
	Title     string
	Path      []string
	StartLine int // Primeira linha de conteúdo PRÓPRIO da seção
	EndLine   int // Onde começa o próximo header (exclusive)
}

// Chunk é a unidade final de indexação.
type Chunk struct {
	ChunkID       string  `json:"chunk_id"`
	ChunkType     string  `json:"chunk_type"` // "command_reference" | "concept"
	Daemon        string  `json:"daemon"`
	Protocol      string  `json:"protocol"`
	SectionPath   string  `json:"section_path"`
	Command       *string `json:"command"`
	Content       string  `json:"content"`
	ParentContent string  `json:"parent_content"`
	SourceURL     string  `json:"source_url"`
	TokenCount    int     `json:"token_count"`
	Order         int     `json:"order"`
}

// HeaderLine é uma ocorrência de header detectada antes da montagem da
// árvore de seções: (linha_do_titulo, nivel, titulo).
type HeaderLine struct {
	TitleLine int
	Level     int
	Title     string
}

// ClicmdEntry é um comando `.. Clicmd::` com seu corpo (descrição/exemplo).
type ClicmdEntry struct {
	Command string
	Body    []string
}

type StackEntry struct {
	Level int
	Title string
}
