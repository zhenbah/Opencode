function logMethods(obj, levels = 3) {
    let proto = obj;
    for (let i = 0; i < levels; i++) {
      proto = Object.getPrototypeOf(proto);
      if (!proto) break;
      const methods = Object.getOwnPropertyNames(proto)
        .filter(name => typeof obj[name] === 'function');
      console.log(`Prototype level ${i + 1} (${proto.constructor.name}):`, methods);
    }
}

export { logMethods }; 