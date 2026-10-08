package verify

import (
	"strings"
	"testing"
)

var (
	png  = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	jpeg = append([]byte("\xff\xd8\xff\xe0"), make([]byte, 64)...)
	webp = []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")
	pdf  = append([]byte("%PDF-1.7\n"), make([]byte, 64)...)
)

func TestSniffType(t *testing.T) {
	for want, data := range map[string][]byte{
		TypePNG: png, TypeJPEG: jpeg, TypeWebP: webp, TypePDF: pdf,
		"": []byte("<script>alert(1)</script>"),
	} {
		if got := SniffType(data); got != want {
			t.Errorf("SniffType = %q, want %q", got, want)
		}
	}
	if SniffType([]byte("RIFF")) != "" {
		t.Error("a short RIFF header is not WebP")
	}
}

func TestCheckUploadMessages(t *testing.T) {
	cases := []struct {
		data     []byte
		allowPDF bool
		want     string
	}{
		{nil, true, "Selfie: the file is empty."},
		{append(png, make([]byte, 1024*1024)...), true, "Selfie: the file is too large (max 1 MB)."},
		{[]byte("<script>"), true, "Selfie: please upload a JPEG, PNG or WebP photo or a PDF."},
		{pdf, false, "Selfie: please upload a JPEG, PNG or WebP photo."},
	}
	for _, c := range cases {
		_, err := CheckUpload(&Upload{Filename: "x", Data: c.data}, "selfie", "Selfie", 1024*1024, c.allowPDF)
		if err == nil || err.Error() != c.want {
			t.Errorf("got %v, want %q", err, c.want)
		}
	}
	doc, err := CheckUpload(&Upload{Filename: "x", Data: pdf}, "council_certificate", "NMC", 1024*1024, true)
	if err != nil || doc.ContentType != TypePDF || doc.Kind != "council_certificate" {
		t.Fatalf("doc %+v, err %v", doc, err)
	}
}

func doctorForm() Form {
	return Form{
		Role: "doctor", FullName: " Dr. Anita Karki ", Phone: "+977 9841234567",
		CitizenshipNumber: "27-01-71-12345", CitizenshipDistrict: "Kathmandu", CouncilNumber: " 12345 ",
		Files: map[string]*Upload{
			"citizenship_front":   {Filename: "front.png", Data: png},
			"citizenship_back":    {Filename: "back.jpg", Data: jpeg},
			"council_certificate": {Filename: "nmc.pdf", Data: pdf},
		},
	}
}

func TestCheckDoctor(t *testing.T) {
	fields, docs, err := Check(doctorForm(), 5<<20)
	if err != nil {
		t.Fatal(err)
	}
	if fields.FullName != "Dr. Anita Karki" || *fields.CouncilNumber != "12345" || fields.Institution != nil {
		t.Fatalf("fields %+v", fields)
	}
	if len(docs) != 3 || docs[2].ContentType != TypePDF {
		t.Fatalf("docs %+v", docs)
	}
}

func TestCheckRefusals(t *testing.T) {
	cases := []struct {
		change func(*Form)
		want   string
	}{
		{func(f *Form) { f.FullName = " A " }, "full name"},
		{func(f *Form) { f.Phone = "abc" }, "phone"},
		{func(f *Form) { f.CitizenshipNumber = "" }, "citizenship number"},
		{func(f *Form) { f.CitizenshipDistrict = " " }, "district"},
		{func(f *Form) { f.Files["citizenship_back"] = &Upload{} }, "both sides"},
		{func(f *Form) { f.Files["citizenship_front"].Data = []byte("<script>") }, "Citizenship (front)"},
		{func(f *Form) { f.CouncilNumber = "" }, "Nepal Medical Council registration number"},
		{func(f *Form) { delete(f.Files, "council_certificate") }, "Nepal Medical Council certificate"},
		{func(f *Form) { f.Role = "nurse"; f.CouncilNumber = "" }, "Nepal Nursing Council"},
		{func(f *Form) { f.Files["selfie"] = &Upload{Filename: "s.pdf", Data: pdf} }, "photo"},
	}
	for _, c := range cases {
		form := doctorForm()
		c.change(&form)
		_, _, err := Check(form, 5<<20)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("got %v, want %q", err, c.want)
		}
	}
}

func TestCheckStudent(t *testing.T) {
	form := Form{
		Role: "student", FullName: "Bikash Thapa", Phone: "9800000000",
		CitizenshipNumber: "45-02-75-00001", CitizenshipDistrict: "Kaski",
		Institution: "Manipal College of Medical Sciences", RecommenderName: "Dr. Anita Karki",
		RecommenderNMC: "12345", CouncilNumber: "ignored",
		Files: map[string]*Upload{
			"citizenship_front": {Filename: "f.png", Data: png},
			"citizenship_back":  {Filename: "b.png", Data: png},
		},
	}
	if _, _, err := Check(form, 5<<20); err == nil || !strings.Contains(err.Error(), "recommendation") {
		t.Fatalf("want the letter to be required, got %v", err)
	}
	form.Files["recommendation_letter"] = &Upload{Filename: "l.pdf", Data: pdf}
	fields, docs, err := Check(form, 5<<20)
	if err != nil || *fields.RecommenderNMC != "12345" || fields.CouncilNumber != nil || len(docs) != 3 {
		t.Fatalf("fields %+v docs %d err %v", fields, len(docs), err)
	}
	form.RecommenderNMC = "!!"
	if _, _, err := Check(form, 5<<20); err == nil || !strings.Contains(err.Error(), "NMC number") {
		t.Fatalf("got %v", err)
	}
}
