/**
 * Landing page and chat copy, in English and Nepali.
 *
 * Each language is written for its readers rather than word for word, and
 * a sentence is shown in one language at a time (the switch is in the
 * header). Product names in the UI (e.g. the "Verified Doctor" badge) stay
 * as they appear in the app.
 */

const en = {
  nav: {
    how: "How it works",
    professionals: "Professionals",
    safety: "Safety",
    talk: "Talk to a professional",
    language: "Language",
  },
  hero: {
    titleA: "First aid now.",
    titleB: "A verified doctor next.",
    lede: "Tell us what happened in English or नेपाली. Get clear first-aid guidance now, and a verified professional when you need more help.",
    primary: "Get first-aid guidance",
    secondary: "Talk to a professional",
    trust: ["English + नेपाली", "Verified professionals", "Built for Nepal"],
    emergencyQ: "Life-threatening emergency?",
    emergencyCall: "Call 102",
  },
  how: {
    title: "One connected path to care.",
    steps: [
      {
        title: "Describe what happened",
        body: "In your own words, in English or नेपाली. No account needed.",
      },
      {
        title: "Get first-aid guidance",
        body: "Short, practical steps for right now. Anything urgent points to 102 first.",
      },
      {
        title: "Connect with a professional",
        body: "A verified doctor, pharmacist or MBBS student joins you on video, with your chat if you choose to share it.",
      },
    ],
  },
  pros: {
    title: "Every professional is reviewed before they meet a patient.",
    body: "Doctors, pharmacists and MBBS students apply with their documents. An administrator checks each application by hand, and patients see who they are talking to on every call.",
    process: ["Apply with documents", "Checked by hand", "Verified on every call"],
    badgeCaption: "What the patient sees during the call",
    requiresTitle: "What each professional provides",
    categories: [
      {
        who: "Doctors",
        needs: "Nepal Medical Council (NMC) registration number and certificate",
      },
      {
        who: "Pharmacists",
        needs: "Nepal Pharmacy Council registration number and certificate",
      },
      {
        who: "MBBS students",
        needs: "A letter of recommendation from a registered doctor, with that doctor's NMC number",
      },
    ],
    everyone: "Everyone also provides their citizenship certificate.",
    apply: "Volunteer as a professional",
  },
  human: {
    lines: ["It is late.", "The clinic is far away.", "You do not know what to do next."],
    answer: "SwasthyaSetu gives you a safe starting point.",
    safetyTitle: "How we keep it safe",
    items: [
      {
        title: "First aid, not a diagnosis",
        body: "The assistant tells you what to do now and when to get help. It does not diagnose or prescribe.",
      },
      {
        title: "Emergencies come first",
        body: "Anything that sounds life-threatening shows 102 straight away, before any AI reply.",
      },
      {
        title: "You decide what to share",
        body: "Your chat reaches a professional only if you choose to share it.",
      },
      {
        title: "Only verified people answer",
        body: "Only professionals an administrator has approved can see waiting patients.",
      },
    ],
  },
  footer: {
    disclaimer: "SwasthyaSetu gives first-aid information, not a medical diagnosis.",
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
    talk: "स्वास्थ्यकर्मीसँग कुरा गर्नुहोस्",
    language: "भाषा",
  },
  hero: {
    titleA: "अहिले प्राथमिक उपचार।",
    titleB: "त्यसपछि प्रमाणित डाक्टर।",
    lede: "के भयो, नेपाली वा अंग्रेजीमा लेख्नुहोस्। तुरुन्तै स्पष्ट प्राथमिक उपचारको निर्देशन पाउनुहोस्, र थप सहयोग चाहिएमा प्रमाणित स्वास्थ्यकर्मीसँग जोडिनुहोस्।",
    primary: "प्राथमिक उपचार निर्देशन पाउनुहोस्",
    secondary: "स्वास्थ्यकर्मीसँग कुरा गर्नुहोस्",
    trust: ["नेपाली + English", "प्रमाणित स्वास्थ्यकर्मी", "नेपालका लागि बनाइएको"],
    emergencyQ: "ज्यान जोखिममा छ?",
    emergencyCall: "102 मा फोन गर्नुहोस्",
  },
  how: {
    title: "हेरचाहसम्म पुग्ने एउटै बाटो।",
    steps: [
      {
        title: "के भयो, बताउनुहोस्",
        body: "आफ्नै शब्दमा, नेपाली वा अंग्रेजीमा। खाता चाहिँदैन।",
      },
      {
        title: "प्राथमिक उपचारको निर्देशन पाउनुहोस्",
        body: "अहिले के गर्ने भन्ने छोटा, व्यावहारिक कदमहरू। जरुरी अवस्थामा सबैभन्दा पहिले 102।",
      },
      {
        title: "स्वास्थ्यकर्मीसँग जोडिनुहोस्",
        body: "प्रमाणित डाक्टर, फार्मासिस्ट वा MBBS विद्यार्थी भिडियोमा जोडिनुहुन्छ। तपाईंले चाहेमा मात्र च्याट देखाइन्छ।",
      },
    ],
  },
  pros: {
    title: "बिरामीसँग भेट्नुअघि हरेक स्वास्थ्यकर्मीको जाँच गरिन्छ।",
    body: "डाक्टर, फार्मासिस्ट र MBBS विद्यार्थीले आफ्ना कागजातसहित आवेदन दिनुहुन्छ। प्रशासकले हरेक आवेदन आफैं जाँच्नुहुन्छ, र हरेक कलमा बिरामीले आफू कोसँग कुरा गर्दै हुनुहुन्छ भन्ने देख्नुहुन्छ।",
    process: ["कागजातसहित आवेदन", "हातैले जाँच", "हरेक कलमा प्रमाणित"],
    badgeCaption: "कलको बेला बिरामीले देख्ने कुरा",
    requiresTitle: "हरेक स्वास्थ्यकर्मीले पेस गर्ने कागजात",
    categories: [
      {
        who: "डाक्टर",
        needs: "नेपाल मेडिकल काउन्सिल (NMC) दर्ता नम्बर र प्रमाणपत्र",
      },
      {
        who: "फार्मासिस्ट",
        needs: "नेपाल फार्मेसी काउन्सिल दर्ता नम्बर र प्रमाणपत्र",
      },
      {
        who: "MBBS विद्यार्थी",
        needs: "दर्ता भएका डाक्टरको सिफारिस पत्र, ती डाक्टरको NMC नम्बरसहित",
      },
    ],
    everyone: "सबैले आफ्नो नागरिकताको प्रमाणपत्र पनि पेस गर्नुहुन्छ।",
    apply: "स्वास्थ्यकर्मीका रूपमा सहयोग गर्नुहोस्",
  },
  human: {
    lines: ["राति अबेर भइसक्यो।", "स्वास्थ्य केन्द्र टाढा छ।", "अब के गर्ने, थाहा छैन।"],
    answer: "स्वास्थ्य सेतुले तपाईंलाई सुरक्षित सुरुआत दिन्छ।",
    safetyTitle: "हामी कसरी सुरक्षित राख्छौं",
    items: [
      {
        title: "प्राथमिक उपचार, निदान होइन",
        body: "सहायकले अहिले के गर्ने र कहिले सहयोग लिने भनेर बताउँछ। यसले रोग निदान गर्दैन, औषधि लेख्दैन।",
      },
      {
        title: "आपत्काल सबैभन्दा पहिले",
        body: "ज्यान जोखिमको संकेत देखिनेबित्तिकै, AI को जवाफ आउनुअघि नै 102 देखाइन्छ।",
      },
      {
        title: "के देखाउने, तपाईं आफैं रोज्नुहुन्छ",
        body: "तपाईंले रोजेमा मात्र तपाईंको च्याट स्वास्थ्यकर्मीसम्म पुग्छ।",
      },
      {
        title: "प्रमाणित व्यक्तिले मात्र जवाफ दिन्छन्",
        body: "प्रशासकले स्वीकृत गरेका स्वास्थ्यकर्मीले मात्र पर्खिरहेका बिरामी देख्न सक्नुहुन्छ।",
      },
    ],
  },
  footer: {
    disclaimer: "स्वास्थ्य सेतुले प्राथमिक उपचारको जानकारी दिन्छ, चिकित्सकीय निदान होइन।",
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
