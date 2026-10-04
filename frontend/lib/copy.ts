/**
 * Landing page and chat copy, in English and Nepali.
 *
 * Each language is written for its readers rather than word for word, and
 * a sentence is shown in one language at a time (the switch is in the
 * header). Every claim here describes what the app actually does: no
 * statistics, testimonials or certifications.
 */

const en = {
  nav: {
    how: "How it works",
    professionals: "Professionals",
    safety: "Safety",
    getHelp: "Get help",
    language: "Language",
  },
  hero: {
    title: ["Help when you're unsure", "what to do next."],
    lede: "First-aid guidance in English or नेपाली, with verified healthcare professionals when you need more help.",
    cta: "Get first-aid help",
    emergencyQ: "Emergency?",
    emergencyCall: "Call 102",
  },
  why: {
    lead: "When the nearest clinic is hours away.",
    title: "Care shouldn't feel far away.",
    body: "It is late, someone is hurt, and you are not sure what to do. SwasthyaSetu gives you a calm first step in your own language, and a way to reach a real professional.",
  },
  how: {
    title: ["First aid now.", "A verified doctor next."],
    steps: [
      {
        title: "Describe",
        body: "Tell us what happened, in your own words, in English or नेपाली. No account needed.",
      },
      {
        title: "Understand what to do first",
        body: "Clear first-aid steps for right now, and a clear call to 102 if it sounds life-threatening.",
      },
      {
        title: "Connect when more help is needed",
        body: "A verified doctor, pharmacist, nurse, paramedic or MBBS student joins you on a video call.",
      },
    ],
    cta: "Get first-aid help",
  },
  pros: {
    title: ["Every professional", "is checked."],
    body: "An administrator reviews every application by hand before anyone can see a waiting patient. On the call, you see who you are talking to.",
    categories: [
      {
        who: "Doctor",
        needs: "Nepal Medical Council registration and certificate",
      },
      {
        who: "Pharmacist",
        needs: "Nepal Pharmacy Council registration and certificate",
      },
      {
        who: "Nurse",
        needs: "Nepal Nursing Council registration and certificate",
      },
      {
        who: "Paramedic",
        needs: "Nepal Health Professional Council registration and certificate",
      },
      {
        who: "MBBS student",
        needs: "Recommendation from a registered doctor",
      },
    ],
    everyone: "Everyone also provides their citizenship certificate.",
    apply: "Volunteer as a professional",
  },
  safety: {
    title: "First-aid information, not a diagnosis.",
    body: "SwasthyaSetu tells you what to do now and when to get help. It does not diagnose, and it does not prescribe. The button to call 102 is always in view, and you decide whether a professional sees your chat.",
  },
  footer: {
    disclaimer: "SwasthyaSetu gives first-aid information, not a medical diagnosis.",
    forProfessionals: "For professionals",
    account: "Account",
  },
  chat: {
    title: "First-aid assistant",
    subtitle: "First-aid information, not a diagnosis",
    emptyTitle: "What happened?",
    emptyBody: "Describe it in your own words, in English or नेपाली.",
    exampleLabel: "Example",
    exampleUser: "I burned my hand while cooking",
    exampleReply:
      "First, cool the burn under cool running water for 20 minutes. Don't use ice or toothpaste.",
    exampleNext: "If you need more help, a verified professional can join you on video.",
    tryLabel: "Or start with",
    examples: [
      "My child has had a fever for 2 days",
      "Someone fell and their ankle is swollen",
      "मौरीले टोक्यो",
    ],
    placeholder: "Describe what happened…",
    message: "Message",
    send: "Send",
    myChats: "My chats",
    newChat: "New chat",
    signInToSave: "Sign in to save chats",
    thinking: "Thinking… the first reply can take up to a minute while the model wakes up.",
    tryAgain: "Try again",
    urgent: "This may be an emergency. Call 102 now, or talk to a professional immediately.",
    call102: "Call 102",
    talk: "Talk to a professional",
  },
};

export type Copy = typeof en;

const ne: Copy = {
  nav: {
    how: "कसरी काम गर्छ",
    professionals: "स्वास्थ्यकर्मी",
    safety: "सुरक्षा",
    getHelp: "सहयोग लिनुहोस्",
    language: "भाषा",
  },
  hero: {
    title: ["के गर्ने भन्ने अलमलमा हुँदा,", "सहयोग यहीँ छ।"],
    lede: "नेपाली वा अंग्रेजीमा प्राथमिक उपचारको निर्देशन, र थप सहयोग चाहिँदा प्रमाणित स्वास्थ्यकर्मी।",
    cta: "प्राथमिक उपचार सहयोग लिनुहोस्",
    emergencyQ: "आपत्काल?",
    emergencyCall: "102 मा फोन गर्नुहोस्",
  },
  why: {
    lead: "जब नजिकको स्वास्थ्य संस्था घण्टौं टाढा हुन्छ।",
    title: "उपचार टाढाको कुरा हुनु हुँदैन।",
    body: "राति अबेर कोही घाइते हुन्छ, र के गर्ने भन्ने थाहा हुँदैन। स्वास्थ्य सेतुले आफ्नै भाषामा शान्त पहिलो कदम देखाउँछ, र साँच्चैको स्वास्थ्यकर्मीसम्म पुग्ने बाटो दिन्छ।",
  },
  how: {
    title: ["अहिले प्राथमिक उपचार।", "त्यसपछि प्रमाणित डाक्टर।"],
    steps: [
      {
        title: "बताउनुहोस्",
        body: "के भयो, आफ्नै शब्दमा, नेपाली वा अंग्रेजीमा लेख्नुहोस्। खाता चाहिँदैन।",
      },
      {
        title: "पहिले के गर्ने, बुझ्नुहोस्",
        body: "अहिलेका लागि प्राथमिक उपचारका स्पष्ट कदम, र ज्यान जोखिमको संकेत भए 102 मा फोन गर्ने स्पष्ट सल्लाह।",
      },
      {
        title: "थप सहयोग चाहिँदा जोडिनुहोस्",
        body: "प्रमाणित डाक्टर, फार्मासिस्ट, नर्स, प्यारामेडिक वा MBBS विद्यार्थी भिडियो कलमा जोडिनुहुन्छ।",
      },
    ],
    cta: "प्राथमिक उपचार सहयोग लिनुहोस्",
  },
  pros: {
    title: ["हरेक स्वास्थ्यकर्मीको", "जाँच हुन्छ।"],
    body: "कुनै पनि स्वास्थ्यकर्मीले पर्खिरहेका बिरामी देख्नुअघि प्रशासकले हरेक आवेदन आफैं जाँच्नुहुन्छ। कलमा, तपाईं कोसँग कुरा गर्दै हुनुहुन्छ भन्ने देख्नुहुन्छ।",
    categories: [
      {
        who: "डाक्टर",
        needs: "नेपाल मेडिकल काउन्सिलको दर्ता र प्रमाणपत्र",
      },
      {
        who: "फार्मासिस्ट",
        needs: "नेपाल फार्मेसी काउन्सिलको दर्ता र प्रमाणपत्र",
      },
      {
        who: "नर्स",
        needs: "नेपाल नर्सिङ काउन्सिलको दर्ता र प्रमाणपत्र",
      },
      {
        who: "प्यारामेडिक",
        needs: "नेपाल स्वास्थ्य व्यवसायी परिषद्को दर्ता र प्रमाणपत्र",
      },
      {
        who: "MBBS विद्यार्थी",
        needs: "दर्ता भएका डाक्टरको सिफारिस",
      },
    ],
    everyone: "सबैले आफ्नो नागरिकताको प्रमाणपत्र पनि पेस गर्नुहुन्छ।",
    apply: "स्वास्थ्यकर्मीका रूपमा सहयोग गर्नुहोस्",
  },
  safety: {
    title: "प्राथमिक उपचारको जानकारी, रोगको निदान होइन।",
    body: "स्वास्थ्य सेतुले अहिले के गर्ने र कहिले सहयोग लिने भनेर बताउँछ। यसले रोग निदान गर्दैन, औषधि पनि लेख्दैन। 102 मा फोन गर्ने बटन सधैं देखिन्छ, र तपाईंको च्याट स्वास्थ्यकर्मीले हेर्ने कि नहेर्ने, तपाईं आफैं रोज्नुहुन्छ।",
  },
  footer: {
    disclaimer: "स्वास्थ्य सेतुले प्राथमिक उपचारको जानकारी दिन्छ, चिकित्सकीय निदान होइन।",
    forProfessionals: "स्वास्थ्यकर्मीका लागि",
    account: "खाता",
  },
  chat: {
    title: "प्राथमिक उपचार सहायक",
    subtitle: "प्राथमिक उपचारको जानकारी, निदान होइन",
    emptyTitle: "के भयो?",
    emptyBody: "आफ्नै शब्दमा, नेपाली वा अंग्रेजीमा लेख्नुहोस्।",
    exampleLabel: "उदाहरण",
    exampleUser: "खाना पकाउँदा मेरो हात पोल्यो",
    exampleReply:
      "पहिले, पोलेको ठाउँलाई २० मिनेटसम्म चिसो बगिरहेको पानीमा राख्नुहोस्। बरफ वा टुथपेस्ट नलगाउनुहोस्।",
    exampleNext: "थप सहयोग चाहिए, प्रमाणित स्वास्थ्यकर्मी भिडियोमा जोडिन सक्नुहुन्छ।",
    tryLabel: "वा यसरी सुरु गर्नुहोस्",
    examples: [
      "मेरो बच्चालाई २ दिनदेखि ज्वरो आएको छ",
      "कोही लडेर गोलीगाँठो सुन्निएको छ",
      "मौरीले टोक्यो",
    ],
    placeholder: "के भयो, लेख्नुहोस्…",
    message: "सन्देश",
    send: "पठाउनुहोस्",
    myChats: "मेरा च्याट",
    newChat: "नयाँ च्याट",
    signInToSave: "च्याट सुरक्षित गर्न साइन इन",
    thinking: "सोच्दै… पहिलो जवाफ आउन एक मिनेटसम्म लाग्न सक्छ।",
    tryAgain: "फेरि प्रयास गर्नुहोस्",
    urgent: "यो आपत्काल हुन सक्छ। अहिले नै 102 मा फोन गर्नुहोस्, वा तुरुन्तै स्वास्थ्यकर्मीसँग कुरा गर्नुहोस्।",
    call102: "102 मा फोन",
    talk: "स्वास्थ्यकर्मीसँग कुरा गर्नुहोस्",
  },
};

export const COPY = { en, ne } as const;
