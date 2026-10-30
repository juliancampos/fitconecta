package professionals

import "testing"

func TestMissingAudienceOther(t *testing.T) {
	catalog := []Audience{
		{ID: 1, Name: "Adultos em geral"},
		{ID: 2, Name: audienceOtherName, IsSpecific: true},
	}

	tests := []struct {
		name  string
		ids   []int
		other string
		want  bool
	}{
		{name: "sem outro publico", ids: []int{1}, other: "", want: false},
		{name: "outro publico sem descricao", ids: []int{1, 2}, other: "", want: true},
		{name: "outro publico com descricao", ids: []int{1, 2}, other: "Atletas masters", want: false},
		{name: "nenhum publico selecionado", ids: nil, other: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := missingAudienceOther(catalog, tt.ids, tt.other); got != tt.want {
				t.Errorf("missingAudienceOther() = %v, want %v", got, tt.want)
			}
		})
	}
}
