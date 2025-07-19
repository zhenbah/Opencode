Prompt Opencode project 

Help me plan out the next step in my project
I am forking the opencode ai agent project, and trying to add new tools to it (in the agent/llm/tools section) and make it even better at running specifically Motion Canvas TS code.  Additionally, and this is very important, I would like to run the opencode agent into a websocket for streaming back to a frontend that is built in vite typescript with a chat application inside. 
Essentially, the user can make a get request from the client using this simple json payload {prompt : hello there ….} using HTTP whilst the backend will need to handle the whole state of the discussion, the agents to run etc. But I am handing off most of these tasks to the opencode agent. I essentially need to plug the agent to the frontend.
The communication from the backend back to the client should be done via a streaming web socket.  Here is the function that I see being very useful to replicate in the @internal/tui/page/chat.go file func (p *chatPage) sendMessage(text string, attachments []message.Attachment) tea.Cmd {
    var cmds []tea.Cmd
    if p.session.ID == "" {
        session, err := p.app.Sessions.Create(context.Background(), "New Session")
        if err != nil {
            return util.ReportError(err)
        }

        p.session = session
        cmd := p.setSidebar()
        if cmd != nil {
            cmds = append(cmds, cmd)
        }
        cmds = append(cmds, util.CmdHandler(chat.SessionSelectedMsg(session)))
    }

    _, err := p.app.CoderAgent.Run(context.Background(), p.session.ID, text, attachments...)
    if err != nil {
        return util.ReportError(err)
    }
    return tea.Batch(cmds...)
}
As you can tell, they have handled an initialization of the CoderAgent. We will need to do this. 
They are also handling session ids 
Can you plan out the entire backend with me step by step. Make choices where they make sense, but if there is a design pattern that requires me to give you more clarity, ask for more guidance on what i actually aim to create. 
Also, for context, the whole job of the backend is to do just like the opencode agent, ie edit code in a specific file but also speak with the user, giving reasoning and discussion

Here is the current structure of the project, where i am placed directly in the opencode main go dir 
.
├── app.log
├── cmd
│   ├── root.go
│   └── schema
├── frontend
│   ├── CHAT_PANEL_README.md
│   ├── chat-test.html
│   ├── node_modules
│   ├── output
│   ├── package.json
│   ├── package-lock.json
│   ├── public
│   ├── src
│   ├── tsconfig.json
│   └── vite.config.ts
├── go.mod
├── go.sum
├── install
├── internal
│   ├── app
│   ├── completions
│   ├── config
│   ├── db
│   ├── diff
│   ├── fileutil
│   ├── format
│   ├── history
│   ├── llm
│   ├── logging
│   ├── lsp
│   ├── message
│   ├── permission
│   ├── pubsub
│   ├── session
│   ├── tui
│   └── version
├── LICENSE
├── logger_tmp.log
├── main
├── main.go
├── main_opencode.txt
├── main_test.go
├── OpenCode.md
├── opencode-schema.json
├── README.md
├── scripts
│   ├── check_hidden_chars.sh
│   ├── release
│   └── snapshot
├── sqlc.yaml
├── test
│   └── integration
├── test_client
├── testing_opencode.log
├── tmp
│   └── main
└── trenaudie_prompts
    └── prompt_plan_backend.md

31 directories, 27 files
