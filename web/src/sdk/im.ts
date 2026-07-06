// 消息类型枚举
export enum MessageType {
    Text = 'text',
    SystemEvent = 'system_event',
    Image = 'image',
    Video = 'video',
    Voice = 'voice',
    Card = 'card',
    Link = 'link',
    Template = 'template',
    File = 'file',
    Location = 'location',
    Ping = 'ping',
    Pong = 'pong'
  }
  
  // 消息状态枚举
  export enum MessageStatus {
    Sending = 'sending',
    Sent = 'sent',
    Delivered = 'delivered',
    Read = 'read',
    Failed = 'failed',
    Recalled = 'recalled'
  }
  
  // SDK 配置接口
  export interface IMSDKOptions {
    baseURL: string;
    wsURL?: string;
    token: string;
  }
  
  // 会话列表最后一条消息快照
  export interface LastMessage {
    msg_id: string
    sequence: number
    sender_id: string
    msg_type: MessageType
    content_preview: string
    client_time: string
  }

  export interface MessageContent {
    text?: string
    mentions?: string[]
    is_at_all?: boolean
    title?: string
    description?: string
    url?: string
    thumb_url?: string
    image_url?: string
    action_url?: string
    price_text?: string
    width?: number
    height?: number
    duration?: number
    size?: number
    format?: string
    md5?: string
    file_name?: string
    latitude?: number
    longitude?: number
    address?: string
    name?: string
    template_id?: string
    items?: Array<Record<string, string>>
    action_text?: string
    event_type?: string
    operator_id?: string
    target_user_ids?: string[]
    group_id?: string
    group_name?: string
    before_value?: string
    after_value?: string
  }

  // 消息接口
  export interface Message {
    id?: string;
    client_message_id?: string;
    conversation_id?: string;
    chat_id?: string;
    seq?: number;
    sender_id?: string;
    sender_profile?: UserInfo;
    type: MessageType;
    content: MessageContent;
    content_preview?: string;
    status?: MessageStatus;
    created_at?: string;
    updated_at?: string;
    /** 服务端通过 WS 推送的接收者会话免打扰状态（仅 WS 消息携带） */
    _ws_muted?: boolean;
    /** 服务端通过 WS 推送的会话业务预览图（仅 WS 消息携带） */
    _ws_preview_image_url?: string;
  }
  
  export interface UserInfo {
    id: string;
    nickname?: string;
    avatar?: string;
    type?: string;
  }

  export interface GroupSummary {
    id: string;
    name: string;
    avatar_url?: string;
    member_count: number;
  }

  export interface Group {
    id: string;
    conversation_id: string;
    name: string;
    avatar_url?: string;
    owner_id: string;
    member_count: number;
    status: string;
    created_at: string;
    updated_at: string;
  }

  export interface GroupMember {
    id: string;
    group_id: string;
    user_id: string;
    role: string;
    status: string;
    group_nickname?: string;
    joined_at: string;
    invited_by?: string;
    user_info?: UserInfo;
  }

  export interface GroupDetailResponse {
    group: Group;
    members?: GroupMember[];
  }

  export interface GroupUpdatePatch {
    name?: string;
  }

  export interface GroupMemberPage {
    items: GroupMember[];
    next_cursor?: string;
    has_more: boolean;
  }

  export interface ConversationReadState {
    conversation_id: string;
    chat_id?: string;
    last_read_sequence: number;
    read_at?: string;
  }

  export interface Conversation {
    id: string;
    conversation_id: string;
    chat_id: string;
    chat_type: string;
    participants: string[];
    last_message: LastMessage;
    display_name?: string;
    display_avatar?: string;
    group_id?: string;
    group_info?: GroupSummary;
    peer_user_info?: UserInfo;
    last_read_sequence: number;
    last_read_at?: string;
    unread_count?: number;
    muted: boolean;
    pinned: boolean;
    preview_image_url?: string;
    created_at: string;
    updated_at: string;
    last_activity: string;
  }

  export interface ConversationPage {
    items: Conversation[];
    next_cursor?: string;
    has_more: boolean;
  }

  export interface ConversationQueryParams {
    limit?: number;
    cursor?: string;
    q?: string;
    active_conversation_id?: string;
  }
  
  // 会话状态接口
  export interface Session {
    user_id: string;
    is_online: boolean;
    last_seen: string;
  }
  
  // 消息查询参数接口
  export interface MessageQueryParams {
    before_id?: string;
    after_id?: string;
    start_time?: string;
    end_time?: string;
    limit?: number;
    cursor?: string;
  }

  export interface MessagePage {
    items: Message[];
    next_cursor?: string;
    has_more: boolean;
  }

  export interface MessageSearchParams {
    q: string;
    limit?: number;
    cursor?: string;
  }

  export interface MessageSearchPage {
    items: Message[];
    next_cursor?: string;
    has_more: boolean;
  }

  export interface ConversationSettingsPatch {
    pinned?: boolean;
    muted?: boolean;
  }

  interface ApiResponse<T> {
    code: number;
    message?: string;
    error?: string;
    data: T;
  }

  const SUCCESS_CODE = 0;

  function unwrapApiResponse<T>(json: unknown): T {
    if (
      json &&
      typeof json === 'object' &&
      'code' in json &&
      typeof (json as { code: unknown }).code === 'number'
    ) {
      const response = json as ApiResponse<T>;
      if (response.code !== SUCCESS_CODE) {
        throw new Error(response.error || response.message || 'Request failed');
      }
      return response.data;
    }

    return json as T;
  }

  function normalizeMessage(raw: Record<string, unknown>): Message {
    let id: string | undefined
    const rawId = raw.message_id ?? raw.msg_id ?? raw.id
    if (rawId != null) {
      if (typeof rawId === 'string') {
        id = rawId
      } else if (typeof rawId === 'object' && rawId !== null && '$oid' in rawId) {
        id = String((rawId as { $oid: string }).$oid)
      } else {
        id = String(rawId)
      }
    }

    const content = normalizeMessageContent(raw.content)

    return {
      id,
      client_message_id: raw.client_message_id == null ? undefined : String(raw.client_message_id),
      conversation_id: raw.conversation_id == null
        ? (raw.chat_id == null ? undefined : String(raw.chat_id))
        : String(raw.conversation_id),
      chat_id: raw.chat_id == null ? undefined : String(raw.chat_id),
      seq: raw.sequence == null
        ? (raw.seq_id == null ? (raw.seq == null ? undefined : Number(raw.seq)) : Number(raw.seq_id))
        : Number(raw.sequence),
      sender_id: String(raw.sender_id ?? ''),
      sender_profile: (raw.sender ?? raw.sender_profile) as UserInfo | undefined,
      type: (raw.message_type ?? raw.msg_type ?? raw.type ?? MessageType.Text) as MessageType,
      content,
      content_preview: raw.content_preview == null ? undefined : String(raw.content_preview),
      status: raw.status as MessageStatus | undefined,
      created_at: raw.created_at as string | undefined,
      updated_at: raw.updated_at as string | undefined,
    };
  }

  function normalizeMessageContent(content: unknown): MessageContent {
    if (content && typeof content === 'object') {
      return content as MessageContent
    }
    if (typeof content === 'string' && content) {
      return { text: content }
    }
    return {}
  }

  function normalizeLastMessage(raw: unknown): LastMessage | undefined {
    if (!raw || typeof raw !== 'object') return undefined
    const item = raw as Record<string, unknown>
    return {
      msg_id: String(item.msg_id ?? item.message_id ?? item.id ?? ''),
      sequence: Number(item.sequence ?? item.seq ?? 0),
      sender_id: String(item.sender_id ?? ''),
      msg_type: (item.msg_type ?? item.message_type ?? item.type ?? MessageType.Text) as MessageType,
      content_preview: item.content_preview == null ? '' : String(item.content_preview),
      client_time: String(item.client_time ?? item.created_at ?? item.server_time ?? new Date().toISOString()),
    }
  }

  function normalizeConversation(raw: Record<string, unknown>): Conversation {
    const conv = raw as unknown as Conversation
    const id = String(raw.conversation_id ?? raw.id ?? conv.id ?? '')
    const lastMessage = normalizeLastMessage(raw.last_message)
    const updatedAt = String(raw.updated_at ?? conv.updated_at ?? new Date().toISOString())
    return {
      ...conv,
      id,
      conversation_id: id,
      chat_id: String(raw.chat_id ?? conv.chat_id ?? ''),
      chat_type: String(raw.chat_type ?? conv.chat_type ?? ''),
      participants: Array.isArray(raw.participants)
        ? raw.participants.map((item) => String(item))
        : (conv.participants ?? []),
      last_message: lastMessage ?? conv.last_message,
      display_name: raw.title == null ? conv.display_name : String(raw.title),
      display_avatar: raw.avatar == null ? conv.display_avatar : String(raw.avatar),
      peer_user_info: (raw.peer_user as UserInfo | null | undefined) ?? conv.peer_user_info,
      last_read_sequence: Number(raw.last_read_sequence ?? conv.last_read_sequence ?? 0),
      last_read_at: raw.last_read_at == null ? conv.last_read_at : String(raw.last_read_at),
      unread_count: raw.unread_count == null && conv.unread_count == null
        ? undefined
        : Number(raw.unread_count ?? conv.unread_count ?? 0),
      muted: Boolean(raw.muted ?? conv.muted ?? false),
      pinned: Boolean(raw.pinned ?? conv.pinned ?? false),
      created_at: String(raw.created_at ?? conv.created_at ?? updatedAt),
      updated_at: updatedAt,
      last_activity: String(raw.last_activity_at ?? conv.last_activity ?? updatedAt),
    }
  }

  async function apiRequest<T>(
    baseURL: string,
    path: string,
    token: string,
    init: RequestInit = {},
  ): Promise<T> {
    const headers: Record<string, string> = {
      Authorization: `Bearer ${token}`,
      ...(init.headers as Record<string, string> | undefined),
    };

    if (init.body && !(init.headers as Record<string, string> | undefined)?.['Content-Type']) {
      headers['Content-Type'] = 'application/json';
    }

    const response = await fetch(`${baseURL}${path}`, {
      ...init,
      headers,
    });

    if (!response.ok) {
      throw new Error(`Request failed: ${response.statusText}`);
    }

    const json = await response.json();
    return unwrapApiResponse<T>(json);
  }
  
  // 连接状态接口
  export interface ConnectionStatus {
    status: 'connected' | 'disconnected' | 'error';
    error?: any;
  }

  interface HeartbeatPing {
    type: MessageType.Ping
    seq_id: number
    client_time: number
  }
  
  // 消息处理器类型
  export type MessageHandler = (message: Message) => void;
  
  // 连接状态处理器类型
  export type ConnectionHandler = (status: ConnectionStatus) => void;
  
  /**
   * IM SDK
   * 封装了 WebSocket 连接、消息发送、接收等功能
   */
  class IMSDK {
    private baseURL: string;
    private wsURL: string;
    private token: string;
    private ws: WebSocket | null;
    private messageHandlers: MessageHandler[];
    private connectionHandlers: ConnectionHandler[];
    private heartbeatInterval: number;
    private heartbeatTimer: ReturnType<typeof setInterval> | null;
    private heartbeatSeq: number;
    private pendingPings: Map<number, number>;
    private maxMissedPongs: number;
    private lastHeartbeatRTT: number | null;
    private clockDiff: number;
    private messageHistoryCursors: Map<string, string | undefined>;
  
    /**
     * 构造函数
     */
    constructor(options: IMSDKOptions) {
      this.baseURL = options.baseURL || (typeof window !== 'undefined' ? window.location.origin : '');
      this.wsURL = options.wsURL || `${this.baseURL.replace(/^http/, 'ws')}/ws`;
      this.token = options.token;
      this.ws = null;
      this.messageHandlers = [];
      this.connectionHandlers = [];
      this.heartbeatInterval = 30000; // 默认30秒发送一次心跳
      this.heartbeatTimer = null;
      this.heartbeatSeq = 0;
      this.pendingPings = new Map();
      this.maxMissedPongs = 3;
      this.lastHeartbeatRTT = null;
      this.clockDiff = 0;
      this.messageHistoryCursors = new Map();
    }
  
    /**
     * 连接 WebSocket
     */
    async connect(): Promise<void> {
      if (this.ws?.readyState === WebSocket.OPEN) {
        return;
      }

      this.ws?.close();

      return new Promise((resolve, reject) => {
        const ws = new WebSocket(`${this.wsURL}?access_token=${encodeURIComponent(this.token)}`);
        this.ws = ws;
        let settled = false;

        const settleResolve = () => {
          if (settled) return;
          settled = true;
          resolve();
        };

        const settleReject = (error: unknown) => {
          if (settled) return;
          settled = true;
          reject(error);
        };
        
        ws.onopen = () => {
          this._notifyConnectionHandlers({ status: 'connected' });
          settleResolve();
        };
        
        ws.onclose = () => {
          if (this.ws === ws) {
            this.ws = null;
            this._notifyConnectionHandlers({ status: 'disconnected' });
          }
          settleReject(new Error('WebSocket closed before connection was established'));
        };
        
        ws.onmessage = (event: MessageEvent) => {
          const raw = JSON.parse(event.data);
          if (raw?.type === MessageType.Pong) {
            this.handlePong(raw as Record<string, unknown>);
            return;
          }
          if (raw?.type === 'message' && raw.data && typeof raw.data === 'object') {
            const data = raw.data as Record<string, unknown>;
            if (!data.message || typeof data.message !== 'object') {
              return;
            }
            const message = normalizeMessage(data.message as Record<string, unknown>);
            const conversation = data.conversation as Record<string, unknown> | undefined;
            if (typeof conversation?.muted === 'boolean') {
              message._ws_muted = conversation.muted;
            }
            if (typeof conversation?.preview_image_url === 'string') {
              message._ws_preview_image_url = conversation.preview_image_url;
            }
            this._notifyMessageHandlers(message);
            return;
          }
          if (raw?.type === MessageType.Ping) {
            return;
          }
          if (raw?.type) {
            return;
          }
          const message = normalizeMessage(raw);
          this._notifyMessageHandlers(message);
        };
        
        ws.onerror = (error: Event) => {
          if (this.ws !== ws) return;
          this._notifyConnectionHandlers({ status: 'error', error });
          settleReject(error);
        };
      });
    }
  
    /**
     * 断开 WebSocket 连接
     */
    disconnect(): void {
      this.stopHeartbeat();
      if (this.ws) {
        this.ws.close();
        this.ws = null;
      }
    }

    isSocketOpen(): boolean {
      return this.ws?.readyState === WebSocket.OPEN;
    }

    updateToken(token: string): void {
      this.token = token;
    }
  
    /**
     * 创建消息
     */
    createMessage(message: Message): Message {
      return {
        ...message,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      }
    }
    /**
     * 通过 WebSocket 发送消息
     */
    async sendMessageWS(
      conversationId: string,
      type: MessageType = MessageType.Text,
      content: MessageContent = {},
      clientMessageId?: string
    ): Promise<void> {
      return new Promise((resolve, reject) => {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
          reject(new Error('WebSocket is not connected'));
          return;
        }
  
        try {
          const message = {
            client_message_id: clientMessageId,
            conversation_id: conversationId,
            message_type: type,
            content,
          };
  
          this.ws.send(JSON.stringify(message));
          resolve();
        } catch (error) {
          reject(new Error(`Failed to send message: ${error}`));
        }
      });
    }
  
    /**
     * 发送消息
     */
    async sendMessage(
      conversationId: string,
      type: MessageType = MessageType.Text,
      content: MessageContent = {},
      clientMessageId?: string
    ): Promise<Message> {
      const response = await fetch(`${this.baseURL}/api/v1/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${this.token}`
        },
        body: JSON.stringify({
          conversation_id: conversationId,
          client_message_id: clientMessageId,
          message_type: type,
          content,
        })
      });

      if (!response.ok) {
        throw new Error(`Failed to send message: ${response.statusText}`);
      }

      const json = await response.json();
      return normalizeMessage(unwrapApiResponse<Record<string, unknown>>(json));
    }
  
    async getConversationMessagePage(conversationId: string, params: MessageQueryParams = {}): Promise<MessagePage> {
      const queryParams = new URLSearchParams();
      if (params.limit) queryParams.append('limit', params.limit.toString());
      if (params.cursor) queryParams.append('cursor', params.cursor);

      const suffix = queryParams.toString() ? `?${queryParams}` : '';
      const data = await apiRequest<Record<string, unknown>[] | Record<string, unknown>>(
        this.baseURL,
        `/api/v1/conversations/${encodeURIComponent(conversationId)}/messages${suffix}`,
        this.token,
      );

      if (Array.isArray(data)) {
        return {
          items: data.map((item) => normalizeMessage(item)),
          has_more: false,
        };
      }

      return {
        items: Array.isArray(data.items)
          ? (data.items as Record<string, unknown>[]).map((item) => normalizeMessage(item))
          : [],
        next_cursor: typeof data.next_cursor === 'string' ? data.next_cursor : undefined,
        has_more: Boolean(data.has_more),
      };
    }

    async getConversationMessages(conversationId: string, params: MessageQueryParams = {}): Promise<Message[]> {
      const cursor = params.cursor
        ?? (params.before_id ? this.messageHistoryCursors.get(conversationId) : undefined);
      const page = await this.getConversationMessagePage(conversationId, {
        limit: params.limit,
        cursor,
      });

      if (!params.after_id) {
        this.messageHistoryCursors.set(conversationId, page.next_cursor);
      }

      return page.items;
    }
  
    /**
     * 获取会话列表
     */
    async getConversationPage(params: ConversationQueryParams = {}): Promise<ConversationPage> {
      const queryParams = new URLSearchParams();
      if (params.limit) queryParams.append('limit', params.limit.toString());
      if (params.cursor) queryParams.append('cursor', params.cursor);
      if (params.q) queryParams.append('q', params.q);
      if (params.active_conversation_id) {
        queryParams.append('active_conversation_id', params.active_conversation_id);
      }

      const suffix = queryParams.toString() ? `?${queryParams}` : '';
      const data = await apiRequest<Record<string, unknown>[] | Record<string, unknown>>(
        this.baseURL,
        `/api/v1/conversations${suffix}`,
        this.token,
      );

      if (Array.isArray(data)) {
        return {
          items: data.map((item) => normalizeConversation(item)),
          has_more: false,
        };
      }

      const items = Array.isArray(data.items)
        ? (data.items as Record<string, unknown>[]).map((item) => normalizeConversation(item))
        : [];

      return {
        items,
        next_cursor: typeof data.next_cursor === 'string' ? data.next_cursor : undefined,
        has_more: Boolean(data.has_more),
      };
    }

    async getConversations(params: ConversationQueryParams = {}): Promise<Conversation[]> {
      const page = await this.getConversationPage(params);
      return page.items;
    }

    async getConversation(conversationId: string): Promise<Conversation> {
      const data = await apiRequest<Record<string, unknown>>(
        this.baseURL,
        `/api/v1/conversations/${encodeURIComponent(conversationId)}`,
        this.token,
      );

      return normalizeConversation(data);
    }

    async activateConversation(peerUserId: string): Promise<Conversation> {
      const data = await apiRequest<Record<string, unknown>>(
        this.baseURL,
        '/api/v1/conversations/single',
        this.token,
        {
          method: 'POST',
          body: JSON.stringify({ peer_user_id: peerUserId }),
        },
      );

      return normalizeConversation(data);
    }

    async getGroup(groupId: string): Promise<GroupDetailResponse> {
      return apiRequest<GroupDetailResponse>(
        this.baseURL,
        `/im/api/groups/${groupId}`,
        this.token,
      );
    }

    async updateGroup(groupId: string, patch: GroupUpdatePatch): Promise<GroupDetailResponse> {
      return apiRequest<GroupDetailResponse>(
        this.baseURL,
        `/im/api/groups/${groupId}`,
        this.token,
        {
          method: 'PATCH',
          body: JSON.stringify(patch),
        },
      );
    }

    async getGroupMembers(
      groupId: string,
      params: { limit?: number; cursor?: string } = {},
    ): Promise<GroupMemberPage> {
      const queryParams = new URLSearchParams();
      if (params.limit) queryParams.append('limit', params.limit.toString());
      if (params.cursor) queryParams.append('cursor', params.cursor);
      const suffix = queryParams.toString() ? `?${queryParams}` : '';
      return apiRequest<GroupMemberPage>(
        this.baseURL,
        `/im/api/groups/${groupId}/members${suffix}`,
        this.token,
      );
    }

    async leaveGroup(groupId: string): Promise<void> {
      await apiRequest<void>(
        this.baseURL,
        `/im/api/groups/${groupId}/leave`,
        this.token,
        { method: 'POST' },
      );
    }

    async markConversationRead(conversationId: string, lastReadSequence?: number): Promise<ConversationReadState> {
      return apiRequest<ConversationReadState>(
        this.baseURL,
        `/api/v1/conversations/${encodeURIComponent(conversationId)}/read`,
        this.token,
        {
          method: 'POST',
          body: JSON.stringify(
            lastReadSequence && lastReadSequence > 0
              ? { last_read_sequence: lastReadSequence }
              : {},
          ),
        },
      );
    }

    async updateConversationSettings(
      conversationId: string,
      settings: ConversationSettingsPatch,
    ): Promise<Conversation> {
      const data = await apiRequest<Record<string, unknown>>(
        this.baseURL,
        `/im/api/conversations/${conversationId}/settings`,
        this.token,
        {
          method: 'PATCH',
          body: JSON.stringify(settings),
        },
      );
      return normalizeConversation(data);
    }

    async searchConversationMessages(
      conversationId: string,
      params: MessageSearchParams,
    ): Promise<MessageSearchPage> {
      const queryParams = new URLSearchParams();
      queryParams.append('q', params.q);
      if (params.limit) queryParams.append('limit', params.limit.toString());
      if (params.cursor) queryParams.append('cursor', params.cursor);
      const data = await apiRequest<{
        items?: Record<string, unknown>[];
        next_cursor?: string;
        has_more?: boolean;
      }>(
        this.baseURL,
        `/im/api/conversations/${conversationId}/messages/search?${queryParams}`,
        this.token,
      );

      return {
        items: (data.items ?? []).map((item) => normalizeMessage(item)),
        next_cursor: data.next_cursor,
        has_more: Boolean(data.has_more),
      };
    }
  
    /**
     * 获取用户在线状态
     */
    async getUserStatus(userID: string): Promise<Session> {
      const data = await apiRequest<Session>(
        this.baseURL,
        `/im/api/sessions/${userID}`,
        this.token,
      );

      return data;
    }
  
    /**
     * 保持在线状态
     */
    async keepAlive(): Promise<void> {
      await apiRequest<void>(
        this.baseURL,
        '/im/api/sessions/keepalive',
        this.token,
        { method: 'POST' },
      );
    }
  
    /**
     * 监听新消息
     */
    onMessage(handler: MessageHandler): void {
      this.messageHandlers.push(handler);
    }
  
    /**
     * 监听连接状态
     */
    onConnection(handler: ConnectionHandler): void {
      this.connectionHandlers.push(handler);
    }
  
    /**
     * 移除消息监听器
     */
    offMessage(handler: MessageHandler): void {
      const index = this.messageHandlers.indexOf(handler);
      if (index > -1) {
        this.messageHandlers.splice(index, 1);
      }
    }
  
    /**
     * 移除连接状态监听器
     */
    offConnection(handler: ConnectionHandler): void {
      const index = this.connectionHandlers.indexOf(handler);
      if (index > -1) {
        this.connectionHandlers.splice(index, 1);
      }
    }
  
    /**
     * 内部方法：通知所有消息处理器
     */
    private _notifyMessageHandlers(message: Message): void {
      this.messageHandlers.forEach(handler => handler(message));
    }
  
    /**
     * 内部方法：通知所有连接状态处理器
     */
    private _notifyConnectionHandlers(status: ConnectionStatus): void {
      this.connectionHandlers.forEach(handler => handler(status));
    }
  
    /**
     * 启动心跳机制
     */
    startHeartbeat(): void {
      if (this.heartbeatTimer) {
        clearInterval(this.heartbeatTimer);
      }
      this.heartbeatSeq = 0;
      this.pendingPings.clear();
  
      this.heartbeatTimer = setInterval(() => {
        this.sendPing().catch((error) => {
          console.error('Heartbeat failed:', error);
          this.disconnect();
        });
      }, this.heartbeatInterval);
    }
  
    /**
     * 停止心跳机制
     */
    stopHeartbeat(): void {
      if (this.heartbeatTimer) {
        clearInterval(this.heartbeatTimer);
        this.heartbeatTimer = null;
      }
      this.pendingPings.clear();
    }

    getHeartbeatRTT(): number | null {
      return this.lastHeartbeatRTT;
    }

    getClockDiff(): number {
      return this.clockDiff;
    }

    private async sendPing(): Promise<void> {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        throw new Error('WebSocket is not connected');
      }

      if (this.pendingPings.size >= this.maxMissedPongs) {
        this.disconnect();
        return;
      }

      const ping: HeartbeatPing = {
        type: MessageType.Ping,
        seq_id: this.heartbeatSeq++,
        client_time: Date.now(),
      };
      this.pendingPings.set(ping.seq_id, ping.client_time);
      this.ws.send(JSON.stringify(ping));
    }

    private handlePong(raw: Record<string, unknown>): void {
      const seqId = Number(raw.seq_id);
      const serverTime = Number(raw.server_time);
      const clientTime = this.pendingPings.get(seqId);
      if (clientTime == null || !Number.isFinite(serverTime)) {
        return;
      }

      this.pendingPings.delete(seqId);

      const now = Date.now();
      const rtt = now - clientTime;
      this.lastHeartbeatRTT = rtt;
      this.clockDiff = serverTime - (clientTime + rtt / 2);
    }
  
    /**
     * 发送消息或心跳
     */
    private async send(message: Message): Promise<void> {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        throw new Error('WebSocket is not connected');
      }
  
      try {
        this.ws.send(JSON.stringify(message));
      } catch (error) {
        console.error('Error sending message:', error);
        throw new Error(`Failed to send message: ${error}`);
      }
    }
  
    /**
     * 发送文本消息的快捷方法
     */
    async sendTextMessage(conversationId: string, content: string): Promise<Message> {
      return this.sendMessage(conversationId, MessageType.Text, { text: content });
    }
  
    /**
     * 发送图片消息的快捷方法
     */
    async sendImageMessage(
      conversationId: string,
      url: string,
      meta?: Record<string, string>
    ): Promise<Message> {
      return this.sendMessage(conversationId, MessageType.Image, {
        url,
        width: meta?.width ? Number(meta.width) : undefined,
        height: meta?.height ? Number(meta.height) : undefined,
        size: meta?.size ? Number(meta.size) : undefined,
        format: meta?.format,
      });
    }
  
    /**
     * 发送视频消息的快捷方法
     */
    async sendVideoMessage(
      conversationId: string,
      url: string,
      meta?: Record<string, string>
    ): Promise<Message> {
      return this.sendMessage(conversationId, MessageType.Video, {
        url,
        width: meta?.width ? Number(meta.width) : undefined,
        height: meta?.height ? Number(meta.height) : undefined,
        duration: meta?.duration ? Number(meta.duration) : undefined,
        size: meta?.size ? Number(meta.size) : undefined,
        format: meta?.format,
      });
    }
  
    /**
     * 发送语音消息的快捷方法
     */
    async sendVoiceMessage(
      conversationId: string,
      url: string,
      meta?: Record<string, string>
    ): Promise<Message> {
      return this.sendMessage(conversationId, MessageType.Voice, {
        url,
        duration: meta?.duration ? Number(meta.duration) : undefined,
        size: meta?.size ? Number(meta.size) : undefined,
        format: meta?.format,
      });
    }
  
    /**
     * 发送卡片消息的快捷方法
     */
    async sendCardMessage(
      conversationId: string,
      title: string,
      description?: string,
      url?: string,
      imageUrl?: string,
      priceText?: string
    ): Promise<Message> {
      return this.sendMessage(conversationId, MessageType.Card, {
        title,
        description,
        action_url: url,
        image_url: imageUrl,
        price_text: priceText,
      });
    }
  
    /**
     * 发送链接消息的快捷方法
     */
    async sendLinkMessage(
      conversationId: string,
      title: string,
      url: string,
      description?: string,
      imageUrl?: string
    ): Promise<Message> {
      return this.sendMessage(conversationId, MessageType.Link, {
        title,
        description,
        url,
        thumb_url: imageUrl,
      });
    }
  }
  
  export default IMSDK; 
