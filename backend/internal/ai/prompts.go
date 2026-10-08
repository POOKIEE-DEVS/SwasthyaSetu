// Package ai talks to MedGemma and keeps its replies safe to show: the
// prompts, the urgent-symptom check, the off-topic filter, and the clean-up
// of replies that contain the model's own reasoning.
package ai

import (
	"regexp"
	"strings"
	"unicode"
)

// Message is one chat turn, as the model server and the browser send it.
type Message struct {
	Role    string `json:"role"` // system | user | assistant
	Content string `json:"content"`
}

// The answer to a request that is plainly not about health (see
// IsOffTopic). Fixed text: the request never reaches the model, so it gets
// nothing to improvise on.
const (
	OffTopicReplyEN = "I can only help with health and first-aid questions. " +
		"If someone is in danger, call 102."
	OffTopicReplyNE = "म स्वास्थ्य र प्राथमिक उपचारसम्बन्धी प्रश्नमा मात्र सहयोग गर्न सक्छु। " +
		"कसैको ज्यान जोखिममा छ भने 102 मा फोन गर्नुहोस्।"
)

// SystemPromptEN is the instruction the model gets, as a few plain
// sentences. Written as a rules list, or with a reminder appended to the
// patient's message, the 4B model treated it as a task to analyse ("The
// user has asked ... Identify the core question ... Plan: ...") and that
// analysis became the reply. Plain prose, like the prompt on the model
// server's own test page, gets plain answers.
const SystemPromptEN = "You are SwasthyaSetu, a first-aid assistant for people in Nepal. Reply in " +
	"English. Give short, practical first-aid steps as a numbered list, under " +
	"180 words. Do not diagnose and never give medicine doses. For anything " +
	"life-threatening, first tell them to call 102 for an ambulance and to " +
	`press "Talk to a professional". You only help with health, illness, ` +
	"injuries and first aid; for anything else, say in one sentence that you " +
	"can only help with health and first-aid questions."

// SystemPromptNE is the same instruction in Nepali, for a message in
// Nepali: that steers the reply into Nepali without asking the model to
// "translate", which it then plans out loud.
const SystemPromptNE = "तपाईं स्वास्थ्य सेतु हुनुहुन्छ, नेपालका मानिसहरूका लागि प्राथमिक उपचार " +
	"सहायक। सधैं नेपाली भाषामा, देवनागरी लिपिमा जवाफ दिनुहोस्। अहिले के गर्ने " +
	"भन्ने छोटा, व्यावहारिक प्राथमिक उपचारका कदमहरू नम्बर लगाएर दिनुहोस्। रोग " +
	"निदान नगर्नुहोस् र औषधिको मात्रा कहिल्यै नबताउनुहोस्। ज्यान जोखिममा देखिए " +
	`सबैभन्दा पहिले 102 मा एम्बुलेन्स बोलाउन र "स्वास्थ्यकर्मीसँग कुरा ` +
	`गर्नुहोस्" थिच्न भन्नुहोस्। स्वास्थ्यसँग सम्बन्धित नभएका प्रश्नमा, तपाईं ` +
	"स्वास्थ्य र प्राथमिक उपचारमा मात्र सहयोग गर्न सक्नुहुन्छ भनेर एक वाक्यमा " +
	"भन्नुहोस्।"

// FallbackSystemPrompt is, word for word, the prompt of the model server's
// test page, which answers well. Used for one retry when a reply still
// comes back as the model's own analysis.
const FallbackSystemPrompt = "You are a first-aid assistant for people in Nepal. Reply in the user's " +
	"language (English or Nepali). Give short, practical first-aid steps. Do " +
	"not diagnose and never give medicine doses. For anything life-threatening, " +
	"tell them to call 102 for an ambulance first."

// A note on regular expressions: Go's \b and \w only know ASCII letters,
// where Python's know every script. The English words below behave the
// same; the Nepali ones are matched without \b, as they always were.

// urgentRE is a keyword check for obvious emergencies. It is deliberately
// crude and conservative: its only job is to surface the "call 102 / talk
// to a doctor" banner immediately, without waiting on the model. It is not
// triage.
var urgentRE = regexp.MustCompile(`(?i)` + strings.Join([]string{
	`chest pain`,
	`heart attack`,
	`can['’]?t breathe`, // phones often type a curly apostrophe
	`cannot breathe`,
	`not breathing`,
	`difficulty breathing`,
	`unconscious`,
	`fainted`,
	`seizure`,
	`\bfits\b`,
	`heavy bleeding`,
	`bleeding a lot`,
	`stroke`,
	`poison`,
	`snake ?bite`,
	`severe burn`,
	`suicid`,
	`छाती दुख`,   // chest pain
	`सास फेर्न`,  // breathing (difficulty)
	`बेहोस`,      // unconscious
	`धेरै रगत`,   // a lot of blood
	`रगत बगि`,    // bleeding
	`छारे`,       // seizure
	`सर्पले टोक`, // snake bite
	`विष खा`,     // ate poison (bare विष would match विषय, "topic")
}, "|"))

// IsUrgent reports whether a message mentions an obvious emergency.
func IsUrgent(text string) bool { return urgentRE.MatchString(text) }

// IsNepali reports whether text contains Devanagari.
func IsNepali(text string) bool {
	for _, r := range text {
		if r >= 0x0900 && r <= 0x097F {
			return true
		}
	}
	return false
}

// MedGemma 1.5 can "think" before answering: <unused94>thought ...
// <unused95>, then the reply. The reasoning is not for patients.
const (
	thoughtStart = "<unused94>"
	thoughtEnd   = "<unused95>"
)

// A thought block whose markers were lost in decoding starts with the bare
// word "thought".
var bareThoughtRE = regexp.MustCompile(`(?i)^\s*thought\s*\n`)

// StripThinking returns only the patient-facing reply. It is empty if the
// model never got past its reasoning (e.g. it ran out of tokens mid-thought).
func StripThinking(text string) string {
	if i := strings.LastIndex(text, thoughtEnd); i >= 0 {
		text = text[i+len(thoughtEnd):]
	} else if strings.HasPrefix(strings.TrimLeftFunc(text, unicode.IsSpace), thoughtStart) {
		return ""
	}
	// There is no telling where a marker-less thought ends, so none of it is
	// shown.
	if bareThoughtRE.MatchString(text) {
		return ""
	}
	return StripPreamble(strings.TrimSpace(text))
}

// With its thinking step skipped, MedGemma sometimes plans out loud in the
// reply instead: "Okay, I understand. I need to provide first-aid advice ...
// in English, using a numbered list, under 180 words ...". That paragraph
// restates our instructions; the patient should only ever see the advice.
var (
	preambleOpenerRE = regexp.MustCompile(`(?i)^\s*(okay|ok|alright|sure|understood|got it|i understand)\b`)
	nepaliOpenerRE   = regexp.MustCompile(`^\s*ठीक छ`)
	preambleMarkerRE = regexp.MustCompile(`(?i)\b(i need to|i should|i must|i will|i'll|i am going to|` +
		`i'm going to|let me|the user|my (task|instructions|response)|numbered list|` +
		`\d+ words|medicine doses|in (english|nepali)|starting directly)\b`)
	paragraphBreakRE = regexp.MustCompile(`\n\s*\n`)
)

func isPreamble(paragraph string) bool {
	if preambleOpenerRE.MatchString(paragraph) {
		return true
	}
	// "ठीक छ" ("okay") as a whole word: not the start of "ठीक छन्".
	if loc := nepaliOpenerRE.FindStringIndex(paragraph); loc != nil {
		rest := []rune(paragraph[loc[1]:])
		if len(rest) == 0 || !isWordRune(rest[0]) {
			return true
		}
	}
	return len(preambleMarkerRE.FindAllStringIndex(paragraph, -1)) >= 2
}

// isWordRune matches Python's \w: letters, numbers and underscore.
func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' }

// StripPreamble drops leading paragraphs in which the model talks about its
// task rather than to the patient. It never empties a reply: a lone
// paragraph stays.
func StripPreamble(text string) string {
	paragraphs := paragraphBreakRE.Split(strings.TrimSpace(text), -1)
	for len(paragraphs) > 1 && isPreamble(paragraphs[0]) {
		paragraphs = paragraphs[1:]
	}
	return strings.TrimSpace(strings.Join(paragraphs, "\n\n"))
}

// BuildModelMessages is the system prompt plus the most recent turns, in a
// shape the Gemma chat template accepts: starting with a user turn and
// strictly alternating. The prompt follows the language of the latest
// message unless one is given. The patient's words go to the model exactly
// as written.
func BuildModelMessages(history []Message, maxMessages int, systemPrompt string) []Message {
	recent := history
	if maxMessages > 0 && len(recent) > maxMessages {
		recent = recent[len(recent)-maxMessages:]
	}
	// Drop leading assistant turns left over from trimming.
	for len(recent) > 0 && recent[0].Role != "user" {
		recent = recent[1:]
	}
	// Merge consecutive same-role turns (e.g. a retried user message).
	var merged []Message
	for _, m := range recent {
		if n := len(merged); n > 0 && merged[n-1].Role == m.Role {
			merged[n-1].Content += "\n\n" + m.Content
			continue
		}
		merged = append(merged, Message{Role: m.Role, Content: m.Content})
	}
	if systemPrompt == "" {
		systemPrompt = SystemPromptEN
		if n := len(merged); n > 0 && IsNepali(merged[n-1].Content) {
			systemPrompt = SystemPromptNE
		}
	}
	return append([]Message{{Role: "system", Content: systemPrompt}}, merged...)
}

// Plainly non-health requests (code, homework, stories, attempts to change
// the rules) get OffTopicReply* without calling the model. Deliberately
// narrow: anything that also mentions a symptom, an injury or care, and
// anything IsUrgent flags, always goes to the model, so this can never
// stand between someone and first aid.
var offTopicRE = regexp.MustCompile(`(?i)` + strings.Join([]string{
	`\b(java|javascript|typescript|python|c\+\+|c#|php|sql|html|css|react|node\.?js|kotlin|rust|golang)\b`,
	`\b(code|coding|programm?ing|program|algorithm|polymorphism|inheritance|compiler?|debug\w*|website|software)\b`,
	`\b(poem|poetry|story|stories|essay|song|lyrics|joke|riddle|novel)\b`,
	`\b(homework|assignment|maths?|mathematics|equation|calculus|algebra)\b`,
	`\b(translate|translation)\b`,
	`\b(bitcoin|crypto\w*|stock market|forex|betting|lottery)\b`,
	`\b(cricket|football|movie|film|netflix|celebrity|election|politic\w*)\b`,
	`\b(hack|hacking|password|phishing)\b`,
	`ignore (all |your |the |previous |above )*(instructions|rules|prompt)`,
	`\b(system prompt|jailbreak|role-?play|pretend to be|act as)\b`,
	`कविता`,    // poem
	`कथा`,      // story
	`गृहकार्य`, // homework
	`गणित`,     // maths
	`चुटकिला`,  // joke
}, "|"))

var healthRE = regexp.MustCompile(`(?i)` + strings.Join([]string{
	`\b(\w*ache\w*|pain\w*|hurt\w*|fever|bleed\w*|blood|burn\w*|cut|wound\w*|injur\w*|fell|fall\w*|` +
		`broke\w*|fracture\w*|swell\w*|swollen|sick|ill|illness|vomit\w*|diarrh\w*|cough\w*|` +
		`breath\w*|chest|head|stomach|pregnan\w*|baby|child|medicine\w*|medication\w*|tablet\w*|` +
		`pill\w*|dose|doctor\w*|hospital|clinic|nurse\w*|poison\w*|bite|bitten|allerg\w*|rash|` +
		`dizz\w*|faint\w*|unconscious|seizure\w*|anxiety|anxious|depress\w*|panic|health\w*|` +
		`first aid|swallow\w*|chok\w*|drown\w*|snake|sting\w*|symptom\w*|disease|infection|eye|` +
		`ear|tooth|teeth)\b`,
	`दुख`,    // pain, hurting
	`ज्वरो`,  // fever
	`रगत`,    // blood
	`पोल`,    // burn
	`चोट`,    // injury
	`बिरामी`, // ill, patient
	`औषधि`,   // medicine
	`डाक्टर`, // doctor
	`बच्चा`,  // child
	`सुन्नि`, // swelling
}, "|"))

// IsOffTopic reports whether a message is plainly not about health.
func IsOffTopic(text string) bool {
	if IsUrgent(text) || healthRE.MatchString(text) {
		return false
	}
	return offTopicRE.MatchString(text)
}

// OffTopicReply is the fixed answer to an off-topic message, in its
// language.
func OffTopicReply(text string) string {
	if IsNepali(text) {
		return OffTopicReplyNE
	}
	return OffTopicReplyEN
}

// Sometimes the model answers with its working instead of the answer: "The
// user has asked ... 1. Identify the core question 2. Assess urgency 3.
// Formulate advice 4. Translate to Nepali". Such a reply is never shown;
// the chat retries once with FallbackSystemPrompt.
var reasoningRE = regexp.MustCompile(`(?im)` + strings.Join([]string{
	`\bthe user(?:'s)? (?:has )?(?:asked|request|is asking|wants|question|message|language|writes)\b`,
	`\bidentify (?:the )?(?:core |main )?(?:question|issue|problem)\b`,
	`\bassess (?:the )?(?:urgency|situation|severity)\b`,
	`\bformulate (?:the )?(?:advice|response|answer|reply)\b`,
	`\btranslate (?:it |this |the advice )?(?:in)?to (?:nepali|english)\b`,
	`^\s*\**plan:?\**\s*$`,
	`\bI should (?:start|begin|reply|respond|provide|tell|mention)\b`,
	`\bmy (?:response|reply|answer) (?:should|will|must)\b`,
}, "|"))

// LooksLikeReasoning reports whether a reply reads as the model's own
// analysis rather than advice.
func LooksLikeReasoning(reply string) bool { return reasoningRE.MatchString(reply) }
